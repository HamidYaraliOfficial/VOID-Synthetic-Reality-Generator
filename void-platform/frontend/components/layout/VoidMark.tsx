'use client';

// VOID's signature mark: a small drifting constellation (nodes + links)
// rendered in the current accent color — the one deliberately playful
// element in an otherwise disciplined Fluent-style shell, evoking a
// "synthetic universe" of connected entities without being literal about it.

export default function VoidMark({ size = 28 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 40 40" fill="none" className="void-constellation" aria-hidden="true">
      <circle cx="20" cy="20" r="18" stroke="var(--border)" strokeWidth="1" opacity="0.5" />
      <line x1="12" y1="14" x2="22" y2="10" stroke="var(--accent)" strokeWidth="1.2" opacity="0.6" />
      <line x1="22" y1="10" x2="29" y2="18" stroke="var(--accent)" strokeWidth="1.2" opacity="0.6" />
      <line x1="12" y1="14" x2="14" y2="27" stroke="var(--accent)" strokeWidth="1.2" opacity="0.45" />
      <line x1="14" y1="27" x2="26" y2="29" stroke="var(--accent)" strokeWidth="1.2" opacity="0.45" />
      <line x1="26" y1="29" x2="29" y2="18" stroke="var(--accent)" strokeWidth="1.2" opacity="0.6" />
      <circle cx="12" cy="14" r="2.6" fill="var(--accent)" />
      <circle cx="22" cy="10" r="2" fill="var(--accent)" opacity="0.85" />
      <circle cx="29" cy="18" r="2.2" fill="var(--accent)" opacity="0.9" />
      <circle cx="14" cy="27" r="1.8" fill="var(--accent)" opacity="0.7" />
      <circle cx="26" cy="29" r="2.4" fill="var(--accent)" />
    </svg>
  );
}
