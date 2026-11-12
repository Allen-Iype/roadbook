#!/bin/sh
# The conversion end of the funnel, from the app stack's database (phase
# 15 CP4, 2026-10-31).
#
#   scripts/pilot/host/accounts.sh [slug]                    (default: app)
#   scripts/pilot/host/accounts.sh --failed-by-day [slug]    (for funnel.py)
#
# funnel.py sees the front — who arrived, who reached sign-in. This sees
# what happened after: one line per account (sign-in day, sessions, last
# seen, imports, candidates, decisions, confirmed) and a totals line, so a
# week reads as landing → signed in → uploaded → confirmed. Email is
# printed because the operator already holds it (it is how the seat was
# given out); nothing here leaves the host. Reads only. The seat cap is
# whatever the stack's env says. `</dev/null` because compose exec
# forwards stdin (the phase 15 CP3 trap).
#
# Failed uploads (issue #12): every upload attempt is a row in `imports`
# with its outcome and the reader's own label for what the file was
# (`detected_format` — stable, where the message is reworded). The read
# ends with the last FAILED_DAYS days of failures as day, kind, accounts,
# attempts — no file name, no email, no message — and the totals line
# carries failed_uploads=. An account whose uploads all fail is otherwise
# invisible here: it shows as "imports 2" and nothing more.
#
# `--failed-by-day` prints only `day|failed attempts` lines for the same
# window, for funnel.py's nightly row, so the query and the compose
# invocation live in one place.
set -eu
FAILED_DAYS=14
MODE=read
if [ "${1:-}" = "--failed-by-day" ]; then MODE=failed; shift; fi
SLUG=${1:-app}
REPO=$(cd "$(dirname "$0")/../../.." && pwd)
PILOT_DIR=${ROADBOOK_PILOT_DIR:-/srv/pilot}
ENV=$PILOT_DIR/instances/$SLUG/.env
[ -f "$ENV" ] || { echo "no instance env at $ENV" >&2; exit 1; }
TZ_NAME=${ROADBOOK_FUNNEL_TZ:-UTC}
[ -f "$PILOT_DIR/front.env" ] && . "$PILOT_DIR/front.env" && TZ_NAME=${ROADBOOK_FUNNEL_TZ:-UTC}

cd "$REPO"
compose() {
  docker compose -p "roadbook-$SLUG" --env-file "$ENV" \
    -f compose.yaml -f scripts/pilot/compose.instance.yaml "$@"
}
if [ "$MODE" = failed ]; then
  echo "select imported_at::date, count(*) from imports
         where status = 'failed' and imported_at >= current_date - $FAILED_DAYS
         group by 1 order by 1;" \
    | compose exec -T -e PGTZ="$TZ_NAME" db psql -U roadbook -d roadbook -At -F'|'
  exit 0
fi
SQL="
select u.email,
       u.created_at::timestamp(0) as signed_in,
       (select count(*) from sessions s where s.user_id=u.id) as sessions,
       (select max(s.created_at)::timestamp(0) from sessions s where s.user_id=u.id) as last_seen,
       (select count(*) from imports i where i.user_id=u.id) as imports,
       (select count(*) from candidates c where c.user_id=u.id) as candidates,
       (select count(*) from decisions d where d.user_id=u.id) as decided,
       (select count(*) from decisions d where d.user_id=u.id and d.action='confirmed') as confirmed
  from users u
 where u.oidc_subject is not null
 order by u.created_at;
select 'TOTAL accounts='||count(*)
       ||' with_upload='||count(*) filter (where exists (select 1 from imports i where i.user_id=u.id))
       ||' with_decision='||count(*) filter (where exists (select 1 from decisions d where d.user_id=u.id))
       ||' with_confirmed='||count(*) filter (where exists (select 1 from decisions d where d.user_id=u.id and d.action='confirmed'))
       ||' failed_uploads='||(select count(*) from imports where status = 'failed')
  from users u where u.oidc_subject is not null;
select 'FAILED UPLOADS, last $FAILED_DAYS days (day | kind | accounts | attempts):';
select imported_at::date, coalesce(detected_format, '(not recognised)'),
       count(distinct user_id), count(*)
  from imports
 where status = 'failed' and imported_at >= current_date - $FAILED_DAYS
 group by 1, 2 order by 1, 2;
"
echo "$SQL" | compose exec -T -e PGTZ="$TZ_NAME" db psql -U roadbook -d roadbook -At -F' | '
echo "(times in $TZ_NAME; seat cap: $(grep -s '^ROADBOOK_USER_CAP=' "$ENV" | cut -d= -f2 || echo 'unset'))"
