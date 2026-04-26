import {
  Component,
  OnInit,
  inject,
  signal,
  computed,
  effect,
  OnDestroy,
  HostListener,
  untracked,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { MatTableModule } from '@angular/material/table';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatTabsModule } from '@angular/material/tabs';
import { MatDialog, MatDialogModule } from '@angular/material/dialog';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { MatMenuModule } from '@angular/material/menu';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatChipsModule } from '@angular/material/chips';
import { BreakpointObserver, Breakpoints } from '@angular/cdk/layout';
import { toSignal } from '@angular/core/rxjs-interop';
import { map } from 'rxjs/operators';
import { firstValueFrom } from 'rxjs';

import { ActionsService, V1Workflow, V1TaskInstance } from '../../generated';
import { UiService } from '../../ui.service';
import { PageHeaderComponent } from '../../shared/page-header.component';
import { RunWorkflowDialogComponent } from './run-workflow-dialog.component';
import { TaskDetailDialogComponent } from './task-detail-dialog.component';
import { CreateWorkflowDialogComponent } from './create-workflow-dialog.component';
import { ConfirmDialogComponent } from '../rbac/confirm-dialog.component';

@Component({
  selector: 'app-actions',
  standalone: true,
  imports: [
    CommonModule,
    MatTableModule,
    MatButtonModule,
    MatIconModule,
    MatTabsModule,
    MatDialogModule,
    MatProgressBarModule,
    MatProgressSpinnerModule,
    MatSlideToggleModule,
    MatMenuModule,
    MatTooltipModule,
    MatChipsModule,
    PageHeaderComponent,
  ],
  templateUrl: './actions.component.html',
  styles: [
    `
      :host {
        display: block;
      }
      .mat-mdc-table {
        background: transparent;
      }
      .mat-mdc-row:hover {
        background-color: var(--mat-sys-surface-container-low);
      }
      .status-chip {
        font-size: 11px;
        font-weight: 700;
        text-transform: uppercase;
        letter-spacing: 0.5px;
      }
    `,
  ],
})
export class ActionsComponent implements OnInit, OnDestroy {
  private orchService = inject(ActionsService);
  public uiService = inject(UiService);
  private dialog = inject(MatDialog);
  private snackBar = inject(MatSnackBar);
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private breakpointObserver = inject(BreakpointObserver);

  isHandset = toSignal(
    this.breakpointObserver.observe(Breakpoints.Handset).pipe(map((result) => result.matches)),
    { initialValue: this.breakpointObserver.isMatched(Breakpoints.Handset) },
  );

  workflows = signal<V1Workflow[]>([]);
  instances = signal<V1TaskInstance[]>([]);

  wfTotal = signal(0);
  wfNextCursor = signal('');
  wfHasMore = signal(false);

  instTotal = signal(0);
  instNextCursor = signal('');
  instHasMore = signal(false);

  pageSize = signal(20);

  selectedWorkflowId = signal<string | null>(null);

  loading = signal(false);
  loadingMore = signal(false);
  selectedTabIndex = signal(0);
  showScrollTop = signal(false);

  displayedWorkflowColumns = computed(() =>
    this.isHandset()
      ? ['enabled', 'name', 'actions']
      : ['enabled', 'name', 'description', 'steps', 'actions'],
  );

  displayedInstanceColumns = computed(() =>
    this.isHandset()
      ? ['status', 'workflowName', 'startedAt', 'actions']
      : ['status', 'id', 'workflowName', 'trigger', 'startedAt', 'actions'],
  );

  private refreshTimer?: any;

  constructor() {
    this.route.queryParams.subscribe((params) => {
      if (params['tab'] === 'instances') {
        this.selectedTabIndex.set(1);
        this.startRefreshTimer();
      } else {
        this.selectedTabIndex.set(0);
        this.stopRefreshTimer();
      }

      const prevWfId = this.selectedWorkflowId();
      const newWfId = params['workflowId'] || null;

      if (newWfId !== prevWfId) {
        this.selectedWorkflowId.set(newWfId);
        untracked(() => this.loadInstances(true));
      }
    });
  }

  ngOnInit(): void {
    this.setupScrollListener();
    this.refreshAll();
  }

  ngOnDestroy(): void {
    this.stopRefreshTimer();
    if (this.scrollListener) {
      const scrollElement = document.querySelector('mat-sidenav-content');
      scrollElement?.removeEventListener('scroll', this.scrollListener);
    }
  }

  private scrollListener?: any;
  private setupScrollListener() {
    const scrollElement = document.querySelector('mat-sidenav-content');
    if (!scrollElement) return;

    this.scrollListener = () => {
      this.showScrollTop.set(scrollElement.scrollTop > 300);
      const atBottom =
        scrollElement.scrollHeight - scrollElement.scrollTop <= scrollElement.clientHeight + 150;

      if (atBottom && !this.loadingMore() && !this.loading()) {
        const tab = this.selectedTabIndex();
        if (tab === 0 && this.wfHasMore()) {
          this.loadWorkflows(false);
        } else if (tab === 1 && this.instHasMore()) {
          this.loadInstances(false, false);
        }
      }
    };
    scrollElement.addEventListener('scroll', this.scrollListener);
  }

  scrollToTop() {
    const scrollElement = document.querySelector('mat-sidenav-content');
    if (scrollElement) {
      scrollElement.scrollTo({ top: 0, behavior: 'smooth' });
    }
  }

  private startRefreshTimer() {
    this.stopRefreshTimer();
    this.refreshTimer = setInterval(() => {
      if (this.selectedTabIndex() === 1 && !this.loading() && !this.loadingMore()) {
        this.loadInstances(true, true);
      }
    }, 10000);
  }

  private stopRefreshTimer() {
    if (this.refreshTimer) {
      clearInterval(this.refreshTimer);
      this.refreshTimer = undefined;
    }
  }

  onTabChange(index: number) {
    this.selectedTabIndex.set(index);
    const tabName = index === 0 ? 'workflows' : 'instances';
    this.router.navigate([], {
      relativeTo: this.route,
      queryParams: { tab: tabName },
      queryParamsHandling: 'merge',
    });

    if (index === 1) {
      this.startRefreshTimer();
      this.loadInstances(true);
    } else {
      this.stopRefreshTimer();
      this.loadWorkflows(true);
    }
  }

  async refreshAll() {
    this.loading.set(true);
    try {
      if (this.selectedTabIndex() === 0) {
        await this.loadWorkflows(true);
      } else {
        await this.loadInstances(true);
      }
    } catch (err) {
      this.snackBar
        .open('加载失败', '重试')
        .onAction()
        .subscribe(() => this.refreshAll());
    } finally {
      this.loading.set(false);
    }
  }

  async loadWorkflows(reset = false) {
    if (reset) {
      this.wfNextCursor.set('');
    } else {
      this.loadingMore.set(true);
    }

    try {
      const res = await firstValueFrom(
        this.orchService.actionsWorkflowsGet(
          this.wfNextCursor(),
          this.pageSize(),
          this.uiService.searchConfig()?.value,
        ),
      );
      const items = (res.items || []) as V1Workflow[];
      if (reset) {
        this.workflows.set(items);
      } else {
        const current = this.workflows();
        const newItems = items.filter((n) => !current.some((e) => e.id === n.id));
        this.workflows.update((prev) => [...prev, ...newItems]);
      }
      this.wfTotal.set(items.length);
      this.wfNextCursor.set(res.nextCursor || '');
      this.wfHasMore.set(res.hasMore || false);
    } finally {
      this.loadingMore.set(false);
    }
  }

  async loadInstances(reset = false, silent = false) {
    if (reset) {
      this.instNextCursor.set('');
      if (!silent) this.loading.set(true);
    } else {
      this.loadingMore.set(true);
    }

    try {
      const res = await firstValueFrom(
        this.orchService.actionsInstancesGet(
          this.instNextCursor(),
          this.pageSize(),
          this.uiService.searchConfig()?.value,
          this.selectedWorkflowId() || undefined,
        ),
      );

      let newItems = (res.items || []) as V1TaskInstance[];

      if (reset) {
        this.instances.set(newItems);
      } else {
        const current = this.instances();
        newItems = newItems.filter((n) => !current.some((e) => e.id === n.id));
        this.instances.update((prev) => [...prev, ...newItems]);
      }

      this.instTotal.set(newItems.length);
      this.instNextCursor.set(res.nextCursor || '');
      this.instHasMore.set(res.hasMore || false);
    } finally {
      this.loadingMore.set(false);
      if (!silent) this.loading.set(false);
    }
  }

  getWorkflowName(id: string | undefined): string {
    if (!id) return '-';
    const wf = this.workflows().find((w) => w.id === id);
    return wf ? wf.meta?.name || id : id;
  }

  isWorkflowRunning(id: string | undefined): boolean {
    if (!id) return false;
    return this.instances().some(
      (i) =>
        i.meta?.workflowId === id &&
        (i.status?.status === 'Running' || i.status?.status === 'Pending'),
    );
  }

  getTriggerLabel(trigger: string | undefined): string {
    if (!trigger) return '-';
    if (trigger === 'Manual') return '手动执行';
    if (trigger === 'Webhook') return 'Webhook';
    if (trigger === 'Cron') return '定时任务';
    return trigger;
  }

  getTriggerIcon(trigger: string | undefined): string {
    switch (trigger) {
      case 'Manual':
        return 'person';
      case 'Webhook':
        return 'link';
      case 'Cron':
        return 'schedule';
      default:
        return 'help_outline';
    }
  }

  getTriggerTooltip(trigger: string | undefined): string {
    return `触发方式: ${this.getTriggerLabel(trigger)}`;
  }

  getStatusClass(status: string | undefined): string {
    const base = 'px-2 py-0.5 rounded text-[10px] font-bold font-mono border ';
    switch (status) {
      case 'Success':
        return base + 'bg-success/10 text-success border-success/20';
      case 'Failed':
        return base + 'bg-error/10 text-error border-error/20';
      case 'Cancelled':
        return base + 'bg-outline/10 text-outline border-outline/20';
      case 'Running':
        return base + 'bg-primary/10 text-primary border-primary/20 animate-pulse';
      case 'Pending':
        return base + 'bg-warning/10 text-warning border-warning/20';
      default:
        return base + 'bg-outline/10 text-outline border-outline/20';
    }
  }

  filterByWorkflow(id: string | null) {
    this.selectedWorkflowId.set(id);
    const tab = id ? 'instances' : this.selectedTabIndex() === 0 ? 'workflows' : 'instances';
    this.router.navigate([], {
      relativeTo: this.route,
      queryParams: { workflowId: id, tab },
      queryParamsHandling: 'merge',
    });
    if (id) {
      this.selectedTabIndex.set(1);
    }
    this.loadInstances(true);
  }

  async toggleWorkflow(workflow: V1Workflow) {
    const updated = {
      ...workflow,
      meta: { ...workflow.meta, enabled: !workflow.meta?.enabled },
    };
    this.loading.set(true);
    try {
      await firstValueFrom(this.orchService.actionsWorkflowsIdPut(workflow.id!, updated));
      this.snackBar.open(updated.meta.enabled ? '工作流已启用' : '工作流已禁用', '了解', {
        duration: 2000,
      });
      await this.loadWorkflows(true);
    } catch (err: any) {
      this.snackBar.open(`操作失败: ${err.error?.message || err.message}`, '了解', {
        duration: 3000,
      });
    } finally {
      this.loading.set(false);
    }
  }

  copyWebhookUrl(workflow: V1Workflow) {
    if (!workflow.meta?.webhookEnabled || !workflow.status?.hasWebhookSecret) {
      return;
    }
    this.snackBar.open('Webhook token 不会在列表接口回传，请先重置 token 后复制。', '了解', {
      duration: 3000,
    });
  }

  async resetWebhookToken(workflow: V1Workflow) {
    this.loading.set(true);
    try {
      await firstValueFrom(this.orchService.actionsWorkflowsIdWebhookResetPost(workflow.id!));
      this.snackBar.open('令牌已重置', '了解', { duration: 2000 });
      await this.loadWorkflows(true);
    } catch (err: any) {
      this.snackBar.open(`重置失败: ${err.error?.message || err.message}`, '了解', {
        duration: 3000,
      });
    } finally {
      this.loading.set(false);
    }
  }

  runWorkflow(workflow: V1Workflow) {
    this.dialog
      .open(RunWorkflowDialogComponent, {
        width: '500px',
        data: { workflow },
      })
      .afterClosed()
      .subscribe(async (res) => {
        if (res) {
          this.loading.set(true);
          try {
            await firstValueFrom(
              this.orchService.actionsWorkflowsWorkflowIdRunPost(workflow.id!, {
                inputs: res,
              }),
            );
            this.snackBar.open('工作流已启动', '了解', { duration: 2000 });
            this.selectedTabIndex.set(1);
            untracked(() => this.loadInstances(true));
          } catch (err: any) {
            this.snackBar.open('启动失败: ' + (err.error?.message || err.message), '了解', {
              duration: 3000,
            });
          } finally {
            this.loading.set(false);
          }
        }
      });
  }

  viewLogs(instance: V1TaskInstance) {
    this.dialog.open(TaskDetailDialogComponent, {
      width: '100vw',
      height: '100vh',
      maxWidth: '100vw',
      maxHeight: '100vh',
      data: { instance },
      panelClass: 'full-screen-dialog',
    });
  }

  createWorkflow() {
    this.dialog
      .open(CreateWorkflowDialogComponent, {
        width: '100vw',
        height: '100vh',
        maxWidth: '100vw',
        maxHeight: '100vh',
        panelClass: 'full-screen-dialog',
      })
      .afterClosed()
      .subscribe((res) => {
        if (res) this.loadWorkflows(true);
      });
  }

  editWorkflow(workflow: V1Workflow) {
    this.dialog
      .open(CreateWorkflowDialogComponent, {
        width: '100vw',
        height: '100vh',
        maxWidth: '100vw',
        maxHeight: '100vh',
        panelClass: 'full-screen-dialog',
        data: { workflow },
      })
      .afterClosed()
      .subscribe((res) => {
        if (res) this.loadWorkflows(true);
      });
  }

  deleteWorkflow(workflow: V1Workflow) {
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      data: {
        title: '删除工作流',
        message: `确定要删除工作流 [${workflow.meta?.name || workflow.id}] 吗？此操作不可撤销。`,
      },
    });

    dialogRef.afterClosed().subscribe(async (result) => {
      if (result) {
        this.loading.set(true);
        try {
          await firstValueFrom(this.orchService.actionsWorkflowsIdDelete(workflow.id!));
          this.snackBar.open('工作流已删除', '了解', { duration: 3000 });
          await this.loadWorkflows(true);
        } catch (err: any) {
          this.snackBar.open(`删除失败: ${err.error?.message || err.message}`, '了解', {
            duration: 5000,
          });
        } finally {
          this.loading.set(false);
        }
      }
    });
  }

  deleteInstance(instance: V1TaskInstance) {
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      data: {
        title: '删除记录',
        message: '确定要删除此条运行记录吗？',
      },
    });

    dialogRef.afterClosed().subscribe(async (result) => {
      if (result) {
        this.loading.set(true);
        try {
          await firstValueFrom(this.orchService.actionsInstancesIdDelete(instance.id!));
          this.snackBar.open('记录已删除', '了解', { duration: 3000 });
          await this.loadInstances(true);
        } catch (err: any) {
          this.snackBar.open(`删除失败: ${err.error?.message || err.message}`, '了解', {
            duration: 5000,
          });
        } finally {
          this.loading.set(false);
        }
      }
    });
  }

  cancelInstance(instance: V1TaskInstance) {
    this.orchService.actionsInstancesIdCancelPost(instance.id!).subscribe({
      next: () => {
        this.snackBar.open('已发送取消请求', '了解', { duration: 2000 });
        this.loadInstances(true, true);
      },
      error: (err) =>
        this.snackBar.open(`取消失败: ${err.error?.message || err.message}`, '了解', {
          duration: 3000,
        }),
    });
  }

  cleanupInstances() {
    this.uiService.openSearch({
      placeholder: '保留天数 (例如 7)',
      value: '7',
      onSearch: async (val: string) => {
        const days = parseInt(val);
        if (isNaN(days)) return;

        this.loading.set(true);
        try {
          const res = await firstValueFrom(this.orchService.actionsInstancesCleanupPost(days));
          this.snackBar.open(`清理成功，删除了 ${res.deleted} 条记录`, '了解', {
            duration: 3000,
          });
          await this.loadInstances(true);
        } catch (err: any) {
          this.snackBar.open(`清理失败: ${err.error?.message || err.message}`, '了解', {
            duration: 5000,
          });
        } finally {
          this.loading.set(false);
        }
      },
    });
  }

  openSearch() {
    this.uiService.openSearch({
      placeholder: this.selectedTabIndex() === 0 ? '搜索工作流...' : '搜索记录...',
      value: this.uiService.searchConfig()?.value || '',
      onSearch: (val: string) => {
        const config = this.uiService.searchConfig();
        if (config) {
          this.uiService.searchConfig.set({ ...config, value: val });
        }
        if (this.selectedTabIndex() === 0) {
          this.loadWorkflows(true);
        } else {
          this.loadInstances(true);
        }
      },
    });
  }

  fabConfig = computed(() => {
    if (this.selectedTabIndex() === 0) {
      return {
        icon: 'add',
        label: '新建工作流',
        action: () => this.createWorkflow(),
      };
    }
    return {
      icon: 'delete_sweep',
      label: '清理记录',
      action: () => this.cleanupInstances(),
    };
  });
}
