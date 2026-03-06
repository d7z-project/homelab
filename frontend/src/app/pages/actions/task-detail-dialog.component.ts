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
import { MatListModule } from '@angular/material/list';
import { MatTooltipModule } from '@angular/material/tooltip';
import { BreakpointObserver, Breakpoints } from '@angular/cdk/layout';
import { toSignal } from '@angular/core/rxjs-interop';
import { map } from 'rxjs/operators';
import {
  ActionsService,
  ModelsTaskInstance,
  ModelsLogEntry,
  ModelsWorkflow,
} from '../../generated';
import { interval, Subscription, firstValueFrom } from 'rxjs';

@Component({
  selector: 'app-task-detail-dialog',
  standalone: true,
  imports: [
    CommonModule,
    MatDialogModule,
    MatButtonModule,
    MatIconModule,
    MatListModule,
    MatTooltipModule,
  ],
  template: `
    <div class="flex flex-col h-full bg-surface overflow-hidden">
      <!-- M3 Standard Header - Optimized for Mobile -->
      <header
        class="flex items-center justify-between px-3 sm:px-6 h-14 sm:h-16 border-b border-outline-variant/20 bg-surface shrink-0"
      >
        <div class="flex items-center gap-2 sm:gap-4 min-w-0 flex-1">
          <div class="flex flex-col min-w-0">
            <div class="flex items-center gap-1.5 sm:gap-2">
              <span
                class="text-[8px] sm:text-[10px] px-1.5 py-0.5 rounded font-bold uppercase tracking-wider bg-primary/10 text-primary border border-primary/20 shrink-0"
                >运行详情</span
              >
              <h2 class="text-xs sm:text-base font-bold tracking-tight m-0 truncate">
                {{ workflowName() }}
              </h2>
            </div>
            <div
              class="flex items-center gap-1.5 sm:gap-2 text-[9px] sm:text-[11px] text-outline mt-0.5 overflow-hidden"
            >
              <mat-icon
                class="!w-2.5 !h-2.5 sm:!w-3 sm:!h-3 !text-[10px] sm:!text-[12px] shrink-0"
                [style.color]="getStatusColor(instance().status)"
                >{{ getStatusIcon(instance().status) }}</mat-icon
              >
              <span class="font-medium uppercase tracking-tighter shrink-0">{{
                instance().status
              }}</span>
              <span class="opacity-30">|</span>
              <span class="truncate">{{ instance().startedAt | date: 'HH:mm:ss' }}</span>
              <span class="opacity-30">|</span>
              <span class="font-mono text-primary/80">{{ duration() }}</span>
            </div>
          </div>
        </div>

        <div class="flex items-center gap-1 sm:gap-2 shrink-0 ml-2">
          @if (instance().status === 'Running') {
            @if (isHandset()) {
              <button
                mat-icon-button
                icon-button-center
                color="warn"
                (click)="cancel()"
                matTooltip="停止执行"
              >
                <mat-icon>stop_circle</mat-icon>
              </button>
            } @else {
              <button mat-button color="warn" (click)="cancel()" class="!rounded-full font-bold">
                <mat-icon>stop_circle</mat-icon>
                停止执行
              </button>
            }
          }
          <div class="w-px h-6 bg-outline-variant/30 mx-1 sm:mx-2"></div>
          <button
            mat-icon-button
            icon-button-center
            (click)="dialogRef.close()"
            matTooltip="关闭详情"
          >
            <mat-icon>close</mat-icon>
          </button>
        </div>
      </header>

      <div class="flex flex-1 flex-col sm:flex-row overflow-hidden">
        <!-- Sidebar / Mobile Steps Scroller -->
        <aside
          [class.w-72]="!isHandset()"
          [class.border-r]="!isHandset()"
          [class.border-b]="isHandset()"
          class="bg-surface-container-low/30 flex flex-col shrink-0 border-outline-variant/10 overflow-hidden"
        >
          <div class="px-6 py-3 hidden sm:flex justify-between items-center shrink-0">
            <span class="text-[10px] font-bold text-outline uppercase tracking-widest"
              >流水线步骤</span
            >
            <span class="text-[9px] px-1.5 py-0.5 rounded bg-outline/10 text-outline font-mono">{{
              stepsList().length + 2
            }}</span>
          </div>

          <!-- Desktop List / Mobile Horizontal Scroll -->
          <div
            class="flex-1 overflow-x-auto sm:overflow-y-auto px-2 sm:px-3 py-2 sm:pb-4 flex sm:flex-col gap-1 no-scrollbar"
          >
            <!-- Initialization -->
            <div
              (click)="onStepChange(0)"
              [class.active-step]="selectedStepIndex() === 0"
              class="step-nav-item group shrink-0 sm:shrink"
            >
              <div class="status-indicator" [style.color]="getStepStatusColor(0)">
                <mat-icon class="!w-4 !h-4 !text-[16px]">{{ getStepStatusIcon(0) }}</mat-icon>
              </div>
              <div class="flex flex-col flex-1 min-w-0">
                <span class="text-[11px] sm:text-xs font-medium truncate">任务初始化</span>
                @if (getStepDuration(0); as d) {
                  <span class="text-[9px] font-mono opacity-50">{{ d }}</span>
                }
              </div>
              <mat-icon class="chevron hidden sm:block opacity-0 group-hover:opacity-100"
                >chevron_right</mat-icon
              >
            </div>

            <!-- Workflow Steps -->
            @for (step of stepsList(); track step.id; let i = $index) {
              <div
                (click)="onStepChange(i + 1)"
                [class.active-step]="selectedStepIndex() === i + 1"
                class="step-nav-item group shrink-0 sm:shrink"
              >
                <div class="status-indicator" [style.color]="getStepStatusColor(i + 1)">
                  <mat-icon class="!w-4 !h-4 !text-[16px]">{{ getStepStatusIcon(i + 1) }}</mat-icon>
                </div>
                <div class="flex flex-col flex-1 min-w-0">
                  <span class="text-[11px] sm:text-xs font-medium truncate">{{
                    step.name || step.id
                  }}</span>
                  @if (getStepDuration(i + 1); as d) {
                    <span class="text-[9px] font-mono opacity-50">{{ d }}</span>
                  }
                </div>
                <mat-icon class="chevron hidden sm:block opacity-0 group-hover:opacity-100"
                  >chevron_right</mat-icon
                >
              </div>
            }

            <!-- Finalization -->
            <div
              (click)="onStepChange(stepsList().length + 1)"
              [class.active-step]="selectedStepIndex() === stepsList().length + 1"
              class="step-nav-item group shrink-0 sm:shrink"
            >
              <div
                class="status-indicator"
                [style.color]="getStepStatusColor(stepsList().length + 1)"
              >
                <mat-icon class="!w-4 !h-4 !text-[16px]">{{
                  getStepStatusIcon(stepsList().length + 1)
                }}</mat-icon>
              </div>
              <div class="flex flex-col flex-1 min-w-0">
                <span class="text-[11px] sm:text-xs font-medium truncate">清理与结束</span>
                @if (getStepDuration(stepsList().length + 1); as d) {
                  <span class="text-[9px] font-mono opacity-50">{{ d }}</span>
                }
              </div>
              <mat-icon class="chevron hidden sm:block opacity-0 group-hover:opacity-100"
                >chevron_right</mat-icon
              >
            </div>
          </div>
        </aside>

        <!-- Main Content: Custom Terminal-like UI -->
        <main class="flex-1 flex flex-col bg-[#1e1e1e] relative min-w-0">
          <!-- Terminal Header (Minimal) -->
          <div
            class="flex items-center justify-between px-4 py-2 bg-[#252526] border-b border-white/5 shrink-0"
          >
            <div class="flex items-center gap-2">
              <mat-icon class="!w-3 !h-3 !text-[12px] text-primary">terminal</mat-icon>
              <span
                class="text-[9px] font-bold font-mono text-white/40 uppercase tracking-widest truncate"
              >
                {{ getStepName(selectedStepIndex()) }}
              </span>
              @if (getStepDuration(selectedStepIndex()); as d) {
                <span class="text-[9px] font-mono text-primary/60 ml-2">[{{ d }}]</span>
              }
            </div>
            <div class="flex items-center gap-3">
              <button
                mat-icon-button
                icon-button-center
                class="!w-6 !h-6 !text-white/30 hover:!text-white/60 transition-colors"
                (click)="autoScroll.set(!autoScroll())"
                [matTooltip]="autoScroll() ? '关闭自动滚动' : '开启自动滚动'"
              >
                <mat-icon class="!text-[14px] !w-4 !h-4">{{
                  autoScroll() ? 'vertical_align_bottom' : 'vertical_align_top'
                }}</mat-icon>
              </button>
            </div>
          </div>

          <!-- Custom Log Viewer -->
          <div
            #scrollContainer
            class="flex-1 overflow-y-auto font-mono text-[13px] leading-relaxed p-4 selection:bg-primary/30 scroll-smooth"
          >
            @for (log of activeLogs(); track $index) {
              <div
                class="flex gap-3 py-0.5 group hover:bg-white/5 rounded px-2 -mx-2 transition-colors"
              >
                <span class="text-white/20 select-none shrink-0 tabular-nums"
                  >[{{ log.timestamp | date: 'HH:mm:ss' }}]</span
                >
                <span class="text-white/80 break-all whitespace-pre-wrap">{{ log.message }}</span>
              </div>
            } @empty {
              <div class="h-full flex items-center justify-center text-white/10 select-none italic">
                等待日志输出...
              </div>
            }
          </div>

          <!-- Error Alert -->
          @if (instance().error && selectedStepIndex() === getCurrentStep()) {
            <div
              class="absolute bottom-4 left-4 right-4 p-3 bg-error/90 backdrop-blur-md rounded-xl text-on-error shadow-2xl flex items-start gap-3 animate-in slide-in-from-bottom-2"
            >
              <mat-icon class="shrink-0 !w-5 !h-5 !text-[20px]">error</mat-icon>
              <div class="flex flex-col gap-0.5">
                <span class="text-[10px] font-bold uppercase tracking-tight">执行错误</span>
                <span class="text-xs opacity-90">{{ instance().error }}</span>
              </div>
            </div>
          }
        </main>
      </div>
    </div>
  `,
  styles: [
    `
      :host {
        display: block;
        width: 100%;
        height: 100%;
        overflow: hidden;
      }
      .step-nav-item {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 8px 12px;
        border-radius: 12px;
        cursor: pointer;
        transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
        color: var(--mat-sys-on-surface-variant);
        white-space: nowrap;
      }
      @media (min-width: 640px) {
        .step-nav-item {
          gap: 12px;
          padding: 10px 16px;
          border-radius: 16px;
          white-space: normal;
        }
      }
      .step-nav-item:hover {
        background-color: var(--mat-sys-surface-container-high);
      }
      .step-nav-item.active-step {
        background-color: var(--mat-sys-primary-container);
        color: var(--mat-sys-on-primary-container);
      }
      .status-indicator {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 20px;
        height: 20px;
        flex-shrink: 0;
      }
      .chevron {
        font-size: 14px;
        width: 14px;
        height: 14px;
        transition: transform 0.2s ease;
      }
      .no-scrollbar::-webkit-scrollbar {
        display: none;
      }
      .no-scrollbar {
        -ms-overflow-style: none;
        scrollbar-width: none;
      }
    `,
  ],
})
export class TaskDetailDialogComponent implements OnInit, OnDestroy {
  @ViewChild('scrollContainer') scrollContainer!: ElementRef;

  private orchService = inject(ActionsService);
  private breakpointObserver = inject(BreakpointObserver);
  private pollSubscription?: Subscription;
  private dialogData = inject(MAT_DIALOG_DATA);

  isHandset = toSignal(
    this.breakpointObserver.observe(Breakpoints.Handset).pipe(map((result) => result.matches)),
    { initialValue: this.breakpointObserver.isMatched(Breakpoints.Handset) },
  );

  instance = signal<ModelsTaskInstance>(this.dialogData.instance);
  workflowName = signal<string>('');
  selectedStepIndex = signal<number>(0);
  stepsList = signal<{ id: string; name: string }[]>([]);
  activeLogs = signal<ModelsLogEntry[]>([]);
  autoScroll = signal(true);
  now = signal(new Date());

  private currentOffset = 0;
  private autoStepFollow = true;

  duration = computed(() => {
    const start = new Date(this.instance().startedAt || new Date()).getTime();
    const end = this.instance().finishedAt
      ? new Date(this.instance().finishedAt!).getTime()
      : this.now().getTime();

    const diff = Math.max(0, Math.floor((end - start) / 1000));
    return this.formatSeconds(diff);
  });

  constructor(public dialogRef: MatDialogRef<TaskDetailDialogComponent>) {
    this.workflowName.set(this.instance().workflowId || 'Unknown Workflow');
    // Effect to handle auto-scrolling when logs change
    effect(() => {
      if (this.activeLogs().length > 0 && this.autoScroll()) {
        this.scrollToBottom();
      }
    });
  }

  async ngOnInit() {
    // Timer for real-time duration updates
    const durationTimer = setInterval(() => {
      this.now.set(new Date());
    }, 1000);

    try {
      const workflows = await firstValueFrom(this.orchService.actionsWorkflowsGet());
      const wf = workflows.find((w) => w.id === this.instance().workflowId);
      if (wf) {
        this.workflowName.set(wf.name || wf.id || '');
        if (wf.steps) {
          this.stepsList.set(wf.steps.map((s) => ({ id: s.id || '', name: s.name || '' })));
        }
      }
    } catch (e) {}

    // Smart initial tab selection
    const inst = this.instance();
    const currentIdx = (inst as any).currentStep ?? 0;
    if (inst.status === 'Running' || inst.status === 'Failed' || inst.status === 'Cancelled') {
      this.selectedStepIndex.set(currentIdx);
    } else {
      this.selectedStepIndex.set(0); // Success -> start from first
    }

    // Initial full fetch for selected step
    this.refreshLogs(true);

    this.pollSubscription = interval(2000).subscribe(() => this.refresh());
    this.pollSubscription.add(() => clearInterval(durationTimer));
  }

  ngOnDestroy() {
    this.pollSubscription?.unsubscribe();
  }

  async refresh() {
    try {
      const insts = await firstValueFrom(this.orchService.actionsInstancesGet());
      const updated = insts.find((i) => i.id === this.instance().id);
      if (updated) {
        this.instance.set(updated);

        // Auto follow step if running using the new backend field
        if (
          this.autoStepFollow &&
          updated.status === 'Running' &&
          (updated as any).currentStep !== undefined
        ) {
          const newIdx = (updated as any).currentStep;
          if (newIdx !== this.selectedStepIndex()) {
            this.onStepChange(newIdx);
            return;
          }
        }

        await this.refreshLogs();
      }
    } catch (e) {}
  }

  onStepChange(index: number) {
    if (this.instance().status !== 'Running') {
      this.autoStepFollow = false;
    }
    this.selectedStepIndex.set(index);
    this.activeLogs.set([]);
    this.currentOffset = 0;
    this.refreshLogs(true);
  }

  getStepName(index: number): string {
    if (index === 0) return '任务初始化';
    if (index === this.stepsList().length + 1) return '清理与结束';
    return this.stepsList()[index - 1]?.name || this.stepsList()[index - 1]?.id || 'Unknown';
  }

  getStepDuration(index: number): string | null {
    const timings = (this.instance() as any).stepTimings;
    if (!timings || !timings[index]) return null;

    const t = timings[index];
    const start = new Date(t.startedAt).getTime();
    const end = t.finishedAt ? new Date(t.finishedAt).getTime() : this.now().getTime();

    const diff = Math.max(0, Math.floor((end - start) / 1000));
    return this.formatSeconds(diff);
  }

  private formatSeconds(diff: number): string {
    const h = Math.floor(diff / 3600);
    const m = Math.floor((diff % 3600) / 60);
    const s = diff % 60;

    if (h > 0) return `${h}h ${m}m ${s}s`;
    if (m > 0) return `${m}m ${s}s`;
    return `${s}s`;
  }

  scrollToBottom() {
    setTimeout(() => {
      if (this.scrollContainer) {
        const el = this.scrollContainer.nativeElement;
        el.scrollTop = el.scrollHeight;
      }
    }, 0);
  }

  async refreshLogs(isInitial = false) {
    const id = this.instance().id;
    if (!id) return;

    try {
      const res = await firstValueFrom<any>(
        this.orchService.actionsInstancesIdLogsGet(
          id,
          this.selectedStepIndex(),
          this.currentOffset,
        ),
      );

      if (res && res.logs) {
        const newLogs: ModelsLogEntry[] = res.logs;
        if (newLogs.length > 0) {
          this.activeLogs.update((prev) => [...prev, ...newLogs]);
          this.currentOffset = res.nextOffset;
        }
      }
    } catch (e) {
      console.error('Failed to fetch logs', e);
    }
  }

  async cancel() {
    const id = this.instance().id;
    if (!id) return;
    try {
      await firstValueFrom(this.orchService.actionsInstancesIdCancelPost(id));
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
        return '#d29922';
      default:
        return '#8b949e';
    }
  }

  getCurrentStep(): number {
    return (this.instance() as any).currentStep ?? -1;
  }

  getStepStatusIcon(index: number): string {
    const status = this.instance().status;
    const currentStep = (this.instance() as any).currentStep ?? -1;

    if (status === 'Success') return 'check_circle';

    if (status === 'Failed' || status === 'Cancelled') {
      if (index === currentStep) return status === 'Failed' ? 'error' : 'cancel';
      if (index < currentStep) return 'check_circle';
      return 'radio_button_unchecked';
    }

    if (status === 'Running') {
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
