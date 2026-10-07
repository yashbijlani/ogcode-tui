import { Show } from 'solid-js';

// Linear-style segmented status circles. Each status reads at a glance:
//   pending      → hollow ring (todo)
//   in_progress  → ring with a ~50% pie fill (working)
//   completed    → solid disc with a check (done)
//   failed       → solid disc with an x (cancelled/failed)
//
// Colors come from the --status-* design tokens (Linear-aligned palette).

const COLOR: Record<string, string> = {
  pending: 'var(--status-pending)',
  in_progress: 'var(--status-progress)',
  completed: 'var(--status-completed)',
  failed: 'var(--status-failed)',
};

export function statusColor(status: string): string {
  return COLOR[status] || COLOR.pending;
}

export default function StatusIcon(props: { status: string; size?: number; class?: string }) {
  const size = () => props.size ?? 16;
  const c = () => statusColor(props.status);

  return (
    <svg
      width={size()}
      height={size()}
      viewBox="0 0 16 16"
      fill="none"
      class={`shrink-0 ${props.class || ''}`}
      aria-label={props.status}
    >
      {/* pending — hollow ring */}
      <Show when={props.status === 'pending'}>
        <circle cx="8" cy="8" r="6" fill="none" stroke={c()} stroke-width="1.5" />
      </Show>

      {/* in_progress — ring + ~50% pie */}
      <Show when={props.status === 'in_progress'}>
        <circle cx="8" cy="8" r="6" fill="none" stroke={c()} stroke-width="1.5" />
        <path d="M8 8 L8 3 A5 5 0 0 1 8 13 Z" fill={c()} />
      </Show>

      {/* completed — solid disc + check */}
      <Show when={props.status === 'completed'}>
        <circle cx="8" cy="8" r="7" fill={c()} />
        <path
          d="M4.6 8.3 l2.1 2.1 l4.7 -4.8"
          fill="none"
          stroke="#fff"
          stroke-width="1.6"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
      </Show>

      {/* failed — solid disc + x */}
      <Show when={props.status === 'failed'}>
        <circle cx="8" cy="8" r="7" fill={c()} />
        <path
          d="M5.6 5.6 l4.8 4.8 M10.4 5.6 l-4.8 4.8"
          stroke="#fff"
          stroke-width="1.6"
          stroke-linecap="round"
        />
      </Show>

      {/* fallback — hollow ring */}
      <Show when={!COLOR[props.status]}>
        <circle cx="8" cy="8" r="6" fill="none" stroke="var(--status-pending)" stroke-width="1.5" />
      </Show>
    </svg>
  );
}
