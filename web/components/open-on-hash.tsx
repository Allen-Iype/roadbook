"use client";

// The one effect on the front door (phase 16 BRIEF §4 CP2): the page folds
// its secondary sections into native <details>, and a rejection link
// (lib/rejection-anchors.ts) may target one of them — "#the-file" or
// "#what-happens-next". A browser scrolls to a fragment but does not open
// the <details> around it, so the reader would land on a closed summary,
// or on nothing visible at all when the target sits inside one. This
// component renders nothing; on load and on every hash change it opens the
// target and every <details> enclosing it, then brings it into view.
//
// Rejected: making the folded sections plain sections that start open
// (the fold is the point — RESEARCH §2 measured the page at eleven hundred
// words); a scripted accordion (the page is deliberately static, phase 7
// DECISIONS 2026-08-11, and <details> already does the work).
import { useEffect } from "react";

export function OpenOnHash() {
  useEffect(() => {
    const reveal = () => {
      const id = decodeURIComponent(window.location.hash.slice(1));
      if (!id) return;
      const target = document.getElementById(id);
      if (!target) return;
      let opened = false;
      for (let el: HTMLElement | null = target; el; el = el.parentElement) {
        if (el instanceof HTMLDetailsElement && !el.open) {
          el.open = true;
          opened = true;
        }
      }
      // The browser's own fragment scroll ran against the closed layout;
      // repeat it now that the target is visible.
      if (opened) target.scrollIntoView();
    };
    reveal();
    window.addEventListener("hashchange", reveal);
    return () => window.removeEventListener("hashchange", reveal);
  }, []);
  return null;
}
