import { Show } from 'solid-js';
import { useNavigate } from '@solidjs/router';
import type { Task } from '../api/client';
import MarkdownContent from './markdown-content';
import { usePlan } from '../context/plan';

const EFFORT_COLORS: Record<string, string> = {
  S: 'bg-emerald-500/15 text-emerald-400 border-emerald-500/20',
  M: 'bg-blue-500/15 text-blue-400 border-blue-500/20',
  L: 'bg-amber-500/15 text-amber-400 border-amber-500/20',
  XL: 'bg-red-500/15 text-red-400 border-red-500/20',
};

const COMPLEXITY_COLORS: Record<string, string> = {
  low: 'text-emerald-400',
  medium: 'text-amber-400',
  high: 'text-red-400',
};

interface TaskCardProps {
  task: Task;
}

export default function TaskCard(props: TaskCardProps) {
  const navigate = useNavigate();
  const plan = usePlan();

  const handleClick = () => {
    if (props.task.status === 'in_progress' || props.task.status === 'completed' || props.task.status === 'failed' || props.task.sessionId) {
      navigate(`/task/${props.task.id}`);
    }
  };

  const isClickable = () => props.task.status !== 'pending' || !!props.task.sessionId;

  return (
    <div
      onClick={handleClick}
      class={`rounded-lg border border-[color:var(--border-default)] bg-[color:var(--bg-surface)]
              p-3 transition-all duration-150 group animate-fade-in
              ${isClickable() ? 'cursor-pointer hover:border-[color:var(--border-strong)]' : 'cursor-default'}`}
    >
      <div class="text-[13px] font-medium text-zinc-200 leading-snug mb-2">{props.task.title}</div>

      <div class="flex items-center gap-1.5 mb-2">
        <span class={`text-[10px] font-medium px-1.5 py-0.5 rounded border ${EFFORT_COLORS[props.task.effort] || EFFORT_COLORS.M}`}>
          {props.task.effort}
        </span>
        <span class={`text-[10px] font-medium ${COMPLEXITY_COLORS[props.task.complexity] || COMPLEXITY_COLORS.medium}`}>
          {props.task.complexity}
        </span>
      </div>

      <Show when={props.task.description}>
        <MarkdownContent text={props.task.description} class="prose-chat-preview line-clamp-2 mb-2 text-zinc-500" />
      </Show>

      <div class="flex items-center gap-2">
        <Show when={props.task.status === 'pending'}>
          <button
            onClick={(e) => { e.stopPropagation(); }}
            class="text-[10px] px-2 py-0.5 rounded border border-[color:var(--border-default)]
                   bg-[color:var(--bg-elevated)] text-zinc-400 hover:text-zinc-200 hover:border-[color:var(--border-strong)]
                   transition-colors duration-150"
          >
            Start
          </button>
        </Show>
        <Show when={props.task.status === 'in_progress'}>
          <button
            onClick={(e) => { e.stopPropagation(); }}
            class="text-[10px] px-2 py-0.5 rounded border border-emerald-500/30
                   bg-emerald-500/10 text-emerald-400 hover:bg-emerald-500/20
                   transition-colors duration-150"
          >
            Complete
          </button>
        </Show>
        <Show when={props.task.status === 'failed'}>
          <button
            onClick={(e) => { e.stopPropagation(); plan.retryTaskById(props.task.id); }}
            class="text-[10px] px-2 py-0.5 rounded border border-red-500/30
                   bg-red-500/10 text-red-400 hover:bg-red-500/20
                   transition-colors duration-150"
          >
            Retry
          </button>
        </Show>
        <Show when={props.task.dependencies.length > 0}>
          <span class="text-[10px] text-zinc-600">
            {props.task.dependencies.length} dep{props.task.dependencies.length > 1 ? 's' : ''}
          </span>
        </Show>
        <div class="flex-1" />
        <Show when={props.task.prUrl}>
          <a
            href={props.task.prUrl!}
            target="_blank"
            rel="noopener noreferrer"
            onClick={(e) => e.stopPropagation()}
            class="text-[10px] text-blue-400 hover:text-blue-300 transition-colors"
          >
            PR #{props.task.prNumber}
          </a>
        </Show>
        <Show when={!props.task.prUrl && props.task.prError}>
          <span
            title={props.task.prError}
            class="text-[10px] text-amber-500 cursor-help"
          >
            No PR
          </span>
        </Show>
      </div>
    </div>
  );
}