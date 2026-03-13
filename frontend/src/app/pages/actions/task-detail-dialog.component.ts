import {
  Component,
  Inject,
  OnInit,
  OnDestroy,
  ViewChild,
  ElementRef,
  inject,
  signal,
  computed,
  effect,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatDialogModule, MatDialogRef, MAT_DIALOG_DATA } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatExpansionModule } from '@angular/material/expansion';
import { BreakpointObserver, Breakpoints } from '@angular/cdk/layout';
import { toSignal } from '@angular/core/rxjs-interop';
import { map } from 'rxjs/operators';
import { ActionsService, ModelsTaskInstance, ModelsLogEntry } from '../../generated';
import { interval, Subscription, firstValueFrom } from 'rxjs';

interface StepState {
  index: number;
  id: string;
  name: string;
  logs: ModelsLogEntry[];
  offset: number;
  expanded: boolean;
  loading: boolean;
}

@Component({
  selector: 'app-task-detail-dialog',
  standalone: true,
  imports: [
    CommonModule,
    MatDialogModule,
    MatButtonModule,
    MatIconModule,
    MatTooltipModule,
    MatExpansionModule,
  ],
  template: `
    <div
      class="flex flex-col h-full bg-surface-container-lowest text-on-surface overflow-hidden font-sans"
    >
      <!-- Header -->
      <header
        class="flex items-center justify-between px-4 sm:px-6 h-16 border-b border-outline-variant/20 bg-surface shrink-0 z-10 shadow-sm"
      >
        <div class="flex items-center gap-4 min-w-0">
          <mat-icon
            class="w-6! h-6! text-[24px]! shrink-0"
            [style.color]="getStatusColor(instance()?.status?.status)"
          >
            {{ getStatusIcon(instance()?.status?.status) }}
          </mat-icon>
          <div class="flex flex-col min-w-0">
            <h2 class="text-sm sm:text-lg font-bold truncate m-0 tracking-tight">
              {{ workflowName() }}
            </h2>
            <div class="flex items-center gap-2 text-[10px] sm:text-xs text-outline font-medium">
              <span class="uppercase tracking-widest">{{
                instance()?.status?.status || 'UNKNOWN'
              }}</span>
              <span class="opacity-30">•</span>
              <span>#{{ instance()?.id?.slice(-6) || 'N/A' }}</span>
              <span class="opacity-30">•</span>
              <span class="text-primary font-mono">{{ duration() }}</span>
            </div>
          </div>
        </div>

        <div class="flex items-center gap-2">
          @if (
            instance()?.status?.status === 'Running' || instance()?.status?.status === 'Pending'
          ) {
            <button mat-button color="warn" (click)="cancel()" class="rounded-full! font-bold">
              停止执行
            </button>
          }
          <button mat-icon-button (click)="dialogRef.close()" class="w-10! h-10! text-outline">
            <mat-icon class="text-[24px]!">close</mat-icon>
          </button>
        </div>
      </header>

      <div class="flex flex-1 overflow-hidden bg-[#fafafa]">
        <!-- Sidebar Navigation -->
        <nav
          class="w-64 border-r border-outline-variant/10 bg-surface hidden lg:flex flex-col shrink-0 py-4 overflow-y-auto custom-scrollbar"
        >
          @for (s of stepStates(); track s.index) {
            <div
              (click)="scrollToStep(s.index)"
              [class.active-nav]="currentRunningStep() === s.index"
              class="px-6 py-3 flex items-center gap-4 cursor-pointer hover:bg-outline/5 transition-all relative group"
            >
              <mat-icon
                class="w-4! h-4! text-[18px]! shrink-0"
                [style.color]="getStepStatusColor(s.index)"
              >
                {{ getStepStatusIcon(s.index) }}
              </mat-icon>
              <div class="flex flex-col min-w-0">
                <span
                  class="text-[13px] truncate font-bold"
                  [class.text-primary]="currentRunningStep() === s.index"
                  [class.text-outline]="currentRunningStep() !== s.index"
                >
                  {{ s.name }}
                </span>
                <span class="text-[9px] font-mono text-outline opacity-60">{{
                  getStepDuration(s.index)
                }}</span>
              </div>
              @if (currentRunningStep() === s.index) {
                <div class="absolute left-0 top-0 bottom-0 w-1 bg-primary"></div>
              }
            </div>
          }
        </nav>

        <!-- Main Scroller -->
        <main #scrollContent class="flex-1 overflow-y-auto scroll-smooth pb-32">
          <div class="max-w-5xl mx-auto p-4 sm:p-8">
            <!-- Unified Pill Container -->
            <div
              class="pill-outer-container shadow-2xl shadow-black/5 bg-surface border border-outline-variant/30 overflow-hidden flex flex-col"
            >
              @for (s of stepStates(); track s.index; let last = $last) {
                <mat-expansion-panel
                  #panel
                  [expanded]="s.expanded"
                  (opened)="onPanelOpened(s.index)"
                  (closed)="s.expanded = false"
                  [class.is-expanded]="s.expanded"
                  class="pill-step-panel shadow-none! bg-transparent! rounded-none! border-none!"
                >
                  <mat-expansion-panel-header class="h-14! px-6! transition-colors duration-200">
                    <mat-panel-title class="flex items-center gap-4 mr-0!">
                      <mat-icon
                        class="w-5! h-5! text-[20px]! shrink-0"
                        [class.animate-spin-slow]="
                          currentRunningStep() === s.index &&
                          getStepStatusIcon(s.index) === 'pending'
                        "
                        [style.color]="getStepStatusColor(s.index)"
                      >
                        {{ getStepStatusIcon(s.index) }}
                      </mat-icon>
                      <span class="text-sm font-bold tracking-tight text-on-surface">{{
                        s.name
                      }}</span>
                      <span
                        class="ml-auto text-[10px] font-mono text-outline opacity-60 bg-outline/5 px-2 py-0.5 rounded-full"
                        >{{ getStepDuration(s.index) }}</span
                      >
                    </mat-panel-title>
                  </mat-expansion-panel-header>

                  <div
                    class="terminal-content font-mono text-[12px] leading-relaxed py-4 bg-[#0d1117] border-y border-outline-variant/10"
                  >
                    @for (log of s.logs; track $index) {
                      <div
                        class="flex gap-4 group hover:bg-white/5 px-6 transition-colors border-l-2 border-transparent hover:border-primary/20"
                      >
                        <span
                          class="text-[#484f58] select-none shrink-0 tabular-nums w-16 text-right opacity-50"
                          >{{ log.timestamp | date: 'HH:mm:ss' }}</span
                        >
                        <span
                          class="break-all whitespace-pre-wrap flex-1 text-[#d1d5db]"
                          [innerHTML]="formatMessage(log.message || '')"
                        ></span>
                      </div>
                    } @empty {
                      <div class="px-6 text-[#484f58] italic py-2">等待日志输出...</div>
                    }
                  </div>
                </mat-expansion-panel>
                @if (!last) {
                  <div class="mx-6 border-b border-outline-variant/10"></div>
                }
              }
            </div>
          </div>
        </main>
      </div>

      <!-- Control Bar -->
      <div class="fixed bottom-10 right-10 flex flex-col gap-3 z-20">
        <button
          mat-mini-fab
          (click)="toggleAutoScroll()"
          [class.!bg-primary]="autoScroll()"
          [class.!bg-surface-container-highest]="!autoScroll()"
          class="text-white! shadow-lg"
          matTooltip="自动跟随"
        >
          <mat-icon>{{ autoScroll() ? 'sync' : 'sync_disabled' }}</mat-icon>
        </button>
        <button
          mat-mini-fab
          (click)="scrollToBottom()"
          class="bg-surface-container-highest! text-on-surface! shadow-lg"
          matTooltip="到底部"
        >
          <mat-icon>vertical_align_bottom</mat-icon>
        </button>
      </div>
    </div>
  `,
  styles: [
    `
      :host {
        display: block;
        width: 100%;
        height: 100%;
        border-radius: 24px;
        overflow: hidden;
      }
      ::ng-deep .task-detail-panel .mat-mdc-dialog-container .mdc-dialog__surface {
        border-radius: 24px !important;
      }
      ::ng-deep .pill-step-panel .mat-expansion-panel-body {
        padding: 0 !important;
      }
      ::ng-deep .pill-step-panel .mat-expansion-indicator {
        display: none !important;
      }

      .pill-outer-container {
        border-radius: 28px;
      }
      .pill-step-panel {
        border: none !important;
        margin: 0 !important;
        box-shadow: none !important;
      }

      /* 展开态：仅改变背景色和左侧线条，绝不显示阴影 */
      .pill-step-panel.is-expanded mat-expansion-panel-header {
        background-color: var(--mat-sys-surface-container-high) !important;
        border-left: 4px solid var(--mat-sys-primary);
      }

      .custom-scrollbar::-webkit-scrollbar {
        width: 4px;
      }
      .custom-scrollbar::-webkit-scrollbar-thumb {
        background: var(--mat-sys-outline-variant);
        border-radius: 10px;
      }

      .animate-spin-slow {
        animation: spin 3s linear infinite;
      }
      @keyframes spin {
        from {
          transform: rotate(0deg);
        }
        to {
          transform: rotate(360deg);
        }
      }

      .active-nav {
        background-color: var(--mat-sys-primary-container) !important;
        color: var(--mat-sys-on-primary-container) !important;
      }
    `,
  ],
})
export class TaskDetailDialogComponent implements OnInit, OnDestroy {
  @ViewChild('scrollContent') scrollContent!: ElementRef;

  private orchService = inject(ActionsService);
  private breakpointObserver = inject(BreakpointObserver);
  private pollSubscription?: Subscription;
  private dialogData = inject(MAT_DIALOG_DATA);

  isHandset = toSignal(
    this.breakpointObserver.observe(Breakpoints.Handset).pipe(map((r) => r.matches)),
    { initialValue: false },
  );
  instance = signal<ModelsTaskInstance | undefined>(this.dialogData?.instance);
  workflowName = signal<string>('');
  stepStates = signal<StepState[]>([]);
  autoScroll = signal(true);
  now = signal(new Date());

  currentRunningStep = computed(() => this.instance()?.status?.currentStep ?? -1);
  private pollingActive = true;
  private lastStepIndex = -1;

  duration = computed(() => {
    const inst = this.instance();
    if (!inst) return '0s';
    const start = new Date(inst.status?.startedAt || new Date()).getTime();
    const end = inst.status?.finishedAt
      ? new Date(inst.status.finishedAt!).getTime()
      : this.now().getTime();
    const diff = Math.max(0, Math.floor((end - start) / 1000));
    return this.formatSeconds(diff);
  });

  constructor(public dialogRef: MatDialogRef<TaskDetailDialogComponent>) {
    const inst = this.instance();
    this.workflowName.set(inst?.meta?.workflowId || 'Workflow');

    effect(() => {
      const inst = this.instance();
      const current = this.currentRunningStep();
      const status = inst?.status?.status;

      // 仅在步骤真正变更且启用了自动跟随，或者刚进入 Running 状态时触发跟随
      if (current !== this.lastStepIndex && current >= 0) {
        if ((status === 'Running' || status === 'Pending') && this.autoScroll()) {
          this.scrollToStep(current);
        }
        this.lastStepIndex = current;
      }
    });
  }

  async ngOnInit() {
    await this.initStepStates();
    // 初始进入时，如果有正在运行的步骤，确保它被记录
    this.lastStepIndex = this.currentRunningStep();

    this.pollSubscription = interval(2000).subscribe(() => {
      this.now.set(new Date());
      this.refresh();
    });
  }

  ngOnDestroy() {
    this.pollingActive = false;
    this.pollSubscription?.unsubscribe();
  }

  private async initStepStates() {
    const inst = this.instance();
    if (!inst) return;

    const steps = (inst.meta?.steps || []).map((s: any) => ({
      id: s.id || '',
      name: s.name || s.id || '',
    }));

    const states: StepState[] = [];
    states.push({
      index: 0,
      id: 'init',
      name: '任务初始化',
      logs: [],
      offset: 0,
      expanded: false,
      loading: false,
    });
    steps.forEach((s: any, i: number) =>
      states.push({
        index: i + 1,
        id: s.id,
        name: s.name,
        logs: [],
        offset: 0,
        expanded: false,
        loading: false,
      }),
    );
    states.push({
      index: steps.length + 1,
      id: 'cleanup',
      name: '清理与结束',
      logs: [],
      offset: 0,
      expanded: false,
      loading: false,
    });

    const status = inst.status?.status;
    const current = this.currentRunningStep();

    if (status === 'Failed' || status === 'Cancelled') {
      const failedIdx = current >= 0 ? current : states.length - 1;
      if (states[failedIdx]) states[failedIdx].expanded = true;
    } else if ((status === 'Running' || status === 'Pending') && current >= 0) {
      if (states[current]) states[current].expanded = true;
    }

    this.stepStates.set(states);
    states.filter((s) => s.expanded).forEach((s) => this.loadLogsForStep(s.index));

    // Async fetch workflow name for display if available, but don't block
    try {
      const wf = await firstValueFrom(
        this.orchService.actionsWorkflowsIdGet(inst.meta?.workflowId!),
      );
      if (wf) this.workflowName.set(wf.meta?.name || wf.id || '');
    } catch (e) {}
  }

  async refresh() {
    if (!this.pollingActive) return;
    const inst = this.instance();
    if (!inst) return;
    try {
      const updated = await firstValueFrom(this.orchService.actionsInstancesIdGet(inst.id!));
      if (updated) {
        this.instance.set(updated);
        const expanded = this.stepStates().filter((s) => s.expanded);
        for (const s of expanded) {
          await this.loadLogsForStep(s.index);
        }
        if (updated.status?.status !== 'Running' && updated.status?.status !== 'Pending') {
          this.pollingActive = false;
        }
      }
    } catch (e) {}
  }

  async onPanelOpened(index: number) {
    const states = this.stepStates();
    const s = states[index];
    if (s) {
      // 如果用户点击的不是当前运行步骤，暂时关闭自动跟随，防止被拉回
      if (this.autoScroll() && index !== this.currentRunningStep()) {
        this.autoScroll.set(false);
      }

      this.expandStepOnly(index);
    }
  }

  private expandStepOnly(index: number) {
    this.stepStates.update((states) => {
      let changed = false;
      states.forEach((s) => {
        const shouldExpand = s.index === index;
        if (s.expanded !== shouldExpand) {
          s.expanded = shouldExpand;
          changed = true;
          if (shouldExpand) {
            this.loadLogsForStep(s.index);
          }
        }
      });
      return changed ? [...states] : states;
    });
  }

  private async loadLogsForStep(index: number) {
    const inst = this.instance();
    if (!inst || !inst.id) return;

    const states = this.stepStates();
    const s = states[index];
    if (!s || s.loading) return;

    s.loading = true;
    try {
      const res = await firstValueFrom<any>(
        this.orchService.actionsInstancesIdLogsGet(inst.id, index, s.offset),
      );
      if (res && res.logs) {
        this.stepStates.update((prevStates) => {
          const target = prevStates[index];
          if (res.logs.length > 0) {
            target.logs = [...target.logs, ...res.logs];
            target.offset = res.nextOffset;
          }
          target.loading = false;
          return [...prevStates];
        });
        // 只有当前展开的步骤是运行中的步骤，且开启了随动，才滚动到底部
        if (res.logs.length > 0 && this.autoScroll() && index === this.currentRunningStep()) {
          this.scrollToBottom();
        }
      } else {
        s.loading = false;
      }
    } catch (e) {
      s.loading = false;
    }
  }

  scrollToStep(index: number) {
    this.expandStepOnly(index);
    setTimeout(() => {
      const panelEl = document.querySelectorAll('.pill-step-panel')[index];
      if (panelEl) {
        panelEl.scrollIntoView({ behavior: 'smooth', block: 'start' });
      }
    }, 150);
  }

  scrollToBottom() {
    setTimeout(() => {
      if (this.scrollContent) {
        const el = this.scrollContent.nativeElement;
        el.scrollTop = el.scrollHeight;
      }
    }, 50);
  }

  toggleAutoScroll() {
    this.autoScroll.set(!this.autoScroll());
    if (this.autoScroll()) {
      this.scrollToBottom();
    }
  }

  formatMessage(msg: string): string {
    let formatted = msg.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
    const ansiMap: { [key: string]: string } = {
      '31': '#f85149',
      '32': '#3fb950',
      '33': '#d29922',
      '34': '#58a6ff',
      '35': '#bc8cff',
      '36': '#39c5bb',
      '90': '#484f58',
    };
    formatted = formatted.replace(/\x1b\[(\d+)m(.*?)\x1b\[0m/g, (match, code, content) => {
      const color = ansiMap[code];
      return color ? `<span style="color: ${color}">${content}</span>` : content;
    });
    return formatted;
  }

  getStepDuration(index: number): string | null {
    const inst = this.instance();
    if (!inst || !inst.status?.stepTimings || !inst.status.stepTimings[index]) return null;
    const t = inst.status.stepTimings[index];
    const start = new Date(t.startedAt!).getTime();
    const end = t.finishedAt ? new Date(t.finishedAt).getTime() : this.now().getTime();
    const diff = Math.max(0, Math.floor((end - start) / 1000));
    return this.formatSeconds(diff);
  }

  private formatSeconds(diff: number): string {
    const m = Math.floor(diff / 60);
    const s = diff % 60;
    return m > 0 ? `${m}m ${s}s` : `${s}s`;
  }

  async cancel() {
    const inst = this.instance();
    if (!inst) return;
    try {
      await firstValueFrom(this.orchService.actionsInstancesIdCancelPost(inst.id!));
      this.refresh();
    } catch (e) {}
  }

  getStatusIcon(status: string | undefined): string {
    switch (status) {
      case 'Success':
        return 'check_circle';
      case 'Failed':
        return 'error';
      case 'Running':
      case 'Pending':
        return 'pending';
      case 'Cancelled':
        return 'cancel';
      default:
        return 'help_outline';
    }
  }

  getStatusColor(status: string | undefined): string {
    switch (status) {
      case 'Success':
        return '#3fb950';
      case 'Failed':
        return '#f85149';
      case 'Running':
      case 'Pending':
        return '#d29922';
      default:
        return '#8b949e';
    }
  }

  getStepStatusIcon(index: number): string {
    const inst = this.instance();
    if (!inst) return 'help_outline';
    const status = inst.status?.status;
    const currentStep = this.currentRunningStep();
    if (status === 'Success') return 'check_circle';
    if (status === 'Failed' || status === 'Cancelled') {
      if (index === currentStep) return status === 'Failed' ? 'error' : 'cancel';
      if (index < currentStep) return 'check_circle';
      return 'radio_button_unchecked';
    }
    if (status === 'Running' || status === 'Pending') {
      if (index === currentStep) return 'pending';
      if (index < currentStep) return 'check_circle';
      return 'radio_button_unchecked';
    }
    return 'help_outline';
  }

  getStepStatusColor(index: number): string {
    const icon = this.getStepStatusIcon(index);
    switch (icon) {
      case 'check_circle':
        return '#3fb950';
      case 'error':
        return '#f85149';
      case 'cancel':
        return '#8b949e';
      case 'pending':
        return '#d29922';
      default:
        return '#8b949e';
    }
  }
}
