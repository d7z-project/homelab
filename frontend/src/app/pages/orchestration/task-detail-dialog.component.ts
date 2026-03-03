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
  OrchestrationService,
  ModelsTaskInstance,
  ModelsLogEntry,
  ModelsWorkflow,
} from '../../generated';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
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
                class="text-[8px] sm:text-[10px] px-1.5 py-0.5 rounded font-mono bg-primary/10 text-primary border border-primary/20 shrink-0"
                >#{{ displayId() }}</span
              >
              <h2 class="text-xs sm:text-base font-bold tracking-tight m-0 truncate">
                {{ instance().workflowId }}
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
            </div>
          </div>
        </div>

        <div class="flex items-center gap-1 sm:gap-2 shrink-0 ml-2">
          @if (instance().status === 'Running') {
            <button
              mat-icon-button
              color="warn"
              (click)="cancel()"
              class="sm:hidden"
              matTooltip="停止执行"
            >
              <mat-icon>stop_circle</mat-icon>
            </button>
            <button
              mat-button
              color="warn"
              (click)="cancel()"
              class="hidden sm:inline-flex !rounded-full font-bold"
            >
              <mat-icon>stop_circle</mat-icon>
              停止执行
            </button>
          }
          <div class="w-px h-6 bg-outline-variant/30 mx-1 sm:mx-2"></div>
          <button mat-icon-button (click)="dialogRef.close()" matTooltip="关闭详情">
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
              stepsList().length + 1
            }}</span>
          </div>

          <!-- Desktop List / Mobile Horizontal Scroll -->
          <div
            class="flex-1 overflow-x-auto sm:overflow-y-auto px-2 sm:px-3 py-2 sm:pb-4 flex sm:flex-col gap-1 no-scrollbar"
          >
            <!-- System Setup -->
            <div
              (click)="onStepChange('')"
              [class.active-step]="selectedStep() === ''"
              class="step-nav-item group shrink-0 sm:shrink"
            >
              <div class="status-indicator">
                <mat-icon class="!w-4 !h-4 !text-[16px]">settings</mat-icon>
              </div>
              <span class="text-[11px] sm:text-xs font-medium truncate flex-1">运行环境</span>
              <mat-icon class="chevron hidden sm:block opacity-0 group-hover:opacity-100"
                >chevron_right</mat-icon
              >
            </div>

            <!-- Workflow Steps -->
            @for (step of stepsList(); track step.id) {
              <div
                (click)="onStepChange(step.id)"
                [class.active-step]="selectedStep() === step.id"
                class="step-nav-item group shrink-0 sm:shrink"
              >
                <div class="status-indicator" [style.color]="getStepStatusColor(step.id)">
                  <mat-icon class="!w-4 !h-4 !text-[16px]">{{
                    getStepStatusIcon(step.id)
                  }}</mat-icon>
                </div>
                <span class="text-[11px] sm:text-xs font-medium truncate flex-1">{{
                  step.name || step.id
                }}</span>
                <mat-icon class="chevron hidden sm:block opacity-0 group-hover:opacity-100"
                  >chevron_right</mat-icon
                >
              </div>
            }
          </div>
        </aside>

        <!-- Main Content: Integrated Terminal -->
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
                {{ selectedStep() === '' ? 'Runner' : 'Step: ' + selectedStep() }}
              </span>
            </div>
          </div>

          <!-- Terminal View -->
          <div class="flex-1 relative overflow-hidden bg-[#1e1e1e]">
            <div #terminalContainer class="h-full w-full"></div>
          </div>

          <!-- Error Alert -->
          @if (instance().error && selectedStep() === '') {
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
      :host ::ng-deep .xterm-viewport {
        overflow-y: auto !important;
        background-color: transparent !important;
      }
      :host ::ng-deep .xterm-screen {
        padding: 16px;
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
  @ViewChild('terminalContainer', { static: true }) terminalContainer!: ElementRef;

  private orchService = inject(OrchestrationService);
  private breakpointObserver = inject(BreakpointObserver);
  private terminal?: Terminal;
  private fitAddon = new FitAddon();
  private pollSubscription?: Subscription;
  private dialogData = inject(MAT_DIALOG_DATA);

  isHandset = toSignal(
    this.breakpointObserver.observe(Breakpoints.Handset).pipe(map((result) => result.matches)),
    { initialValue: this.breakpointObserver.isMatched(Breakpoints.Handset) },
  );

  instance = signal<ModelsTaskInstance>(this.dialogData.instance);
  selectedStep = signal<string>('');

  displayId = computed(() => {
    const id = this.instance().id || '';
    return id.split('_').pop() || id;
  });

  stepsList = signal<{ id: string; name: string }[]>([]);

  private lastRenderedIndex = -1;

  constructor(public dialogRef: MatDialogRef<TaskDetailDialogComponent>) {}

  async ngOnInit() {
    try {
      const workflows = await firstValueFrom(this.orchService.orchestrationWorkflowsGet());
      const wf = workflows.find((w) => w.id === this.instance().workflowId);
      if (wf && wf.steps) {
        this.stepsList.set(wf.steps.map((s) => ({ id: s.id || '', name: s.name || '' })));
      }
    } catch (e) {}

    this.terminal = new Terminal({
      theme: {
        background: '#1e1e1e',
        foreground: '#d4d4d4',
        cursor: '#aeafad',
        selectionBackground: '#264f78',
        black: '#000000',
        red: '#f44747',
        green: '#6a9955',
        yellow: '#d7ba7d',
        blue: '#569cd6',
        magenta: '#c586c0',
        cyan: '#9cdcfe',
        white: '#d4d4d4',
      },
      convertEol: true,
      disableStdin: true,
      fontSize: 12,
      fontFamily: "'JetBrains Mono', 'Roboto Mono', 'Menlo', 'Monaco', monospace",
      lineHeight: 1.5,
      letterSpacing: 0,
    });
    this.terminal.loadAddon(this.fitAddon);
    this.terminal.open(this.terminalContainer.nativeElement);

    // Fit terminal on open and window resize
    setTimeout(() => this.fitAddon.fit(), 50);
    window.addEventListener('resize', () => this.fitAddon.fit());

    this.renderLogs();

    if (this.instance().status === 'Running') {
      this.pollSubscription = interval(2000).subscribe(() => this.refresh());
    }
  }

  ngOnDestroy() {
    this.pollSubscription?.unsubscribe();
    this.terminal?.dispose();
    window.removeEventListener('resize', () => this.fitAddon.fit());
  }

  async refresh() {
    try {
      const insts = await firstValueFrom(this.orchService.orchestrationInstancesGet());
      const updated = insts.find((i) => i.id === this.instance().id);
      if (updated) {
        this.instance.set(updated);
        this.renderLogs();
        if (updated.status !== 'Running') {
          this.pollSubscription?.unsubscribe();
        }
      }
    } catch (e) {}
  }

  onStepChange(step: string) {
    this.selectedStep.set(step);
    this.clearTerminal();
    this.lastRenderedIndex = -1;
    this.renderLogs();
    setTimeout(() => this.fitAddon.fit(), 0);
  }

  clearTerminal() {
    this.terminal?.clear();
    this.terminal?.write('\x1b[2J\x1b[H');
  }

  renderLogs() {
    const allLogs = this.instance().logs || [];
    const step = this.selectedStep();

    const startIndex = this.lastRenderedIndex + 1;

    for (let i = startIndex; i < allLogs.length; i++) {
      const log = allLogs[i];
      if (log.stepId === step || (!log.stepId && step === '')) {
        const time = new Date(log.timestamp || '').toLocaleTimeString([], { hour12: false });
        this.terminal?.write(`\x1b[90m${time}\x1b[0m  ${log.message}\r\n`);
      }
      this.lastRenderedIndex = i;
    }
  }

  async cancel() {
    const id = this.instance().id;
    if (!id) return;
    try {
      await firstValueFrom(this.orchService.orchestrationInstancesIdCancelPost(id));
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

  getStepStatusIcon(stepId: string): string {
    const logs = this.instance().logs || [];
    const hasLogs = logs.some((l) => l.stepId === stepId);
    if (!hasLogs) return 'radio_button_unchecked';

    if (this.instance().status === 'Success') return 'check_circle';
    if (this.instance().status === 'Failed') {
      const lastLog = logs[logs.length - 1];
      if (lastLog.stepId === stepId) return 'error';
      return 'check_circle';
    }
    if (this.instance().status === 'Running') {
      const lastLog = logs[logs.length - 1];
      if (lastLog.stepId === stepId) return 'pending';
      return 'check_circle';
    }
    return 'check_circle';
  }

  getStepStatusColor(stepId: string): string {
    const icon = this.getStepStatusIcon(stepId);
    switch (icon) {
      case 'check_circle':
        return '#3fb950';
      case 'error':
        return '#f85149';
      case 'pending':
        return '#d29922';
      default:
        return '#8b949e';
    }
  }
}
