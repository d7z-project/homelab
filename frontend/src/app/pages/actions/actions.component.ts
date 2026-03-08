import {
  Component,
  OnInit,
  inject,
  signal,
  computed,
  effect,
  OnDestroy,
  HostListener,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { MatTableModule } from '@angular/material/table';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatCardModule } from '@angular/material/card';
import { MatTabsModule } from '@angular/material/tabs';
import { MatDividerModule } from '@angular/material/divider';
import { ActionsService, ModelsWorkflow, ModelsTaskInstance } from '../../generated';
import { MatDialog, MatDialogModule } from '@angular/material/dialog';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatMenuModule } from '@angular/material/menu';
import { MatChipsModule } from '@angular/material/chips';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { firstValueFrom } from 'rxjs';
import { BreakpointObserver, Breakpoints } from '@angular/cdk/layout';
import { Clipboard } from '@angular/cdk/clipboard';
import { toSignal } from '@angular/core/rxjs-interop';
import { map } from 'rxjs/operators';
import { UiService } from '../../ui.service';
import { ConfirmDialogComponent } from '../rbac/confirm-dialog.component';
import { CreateWorkflowDialogComponent } from './create-workflow-dialog.component';
import { RunWorkflowDialogComponent } from './run-workflow-dialog.component';
import { TaskDetailDialogComponent } from './task-detail-dialog.component';
import { PageHeaderComponent } from '../../shared/page-header.component';

@Component({
  selector: 'app-actions',
  standalone: true,
  imports: [
    CommonModule,
    MatTableModule,
    MatButtonModule,
    MatIconModule,
    MatCardModule,
    MatDialogModule,
    MatTabsModule,
    MatDividerModule,
    MatProgressBarModule,
    MatProgressSpinnerModule,
    MatTooltipModule,
    MatMenuModule,
    MatChipsModule,
    MatSlideToggleModule,
    PageHeaderComponent,
  ],
  templateUrl: './actions.component.html',
})
export class ActionsComponent implements OnInit, OnDestroy {
  private orchService = inject(ActionsService);
  private snackBar = inject(MatSnackBar);
  private dialog = inject(MatDialog);
  private clipboard = inject(Clipboard);
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private breakpointObserver = inject(BreakpointObserver);
  public uiService = inject(UiService);

  isHandset = toSignal(
    this.breakpointObserver.observe(Breakpoints.Handset).pipe(map((result) => result.matches)),
    { initialValue: this.breakpointObserver.isMatched(Breakpoints.Handset) },
  );

  workflows = signal<ModelsWorkflow[]>([]);
  instances = signal<ModelsTaskInstance[]>([]);

  wfTotal = signal(0);
  wfNextCursor = signal('');
  wfHasMore = signal(false);

  instTotal = signal(0);
  instNextCursor = signal('');
  instHasMore = signal(false);

  selectedWorkflowId = signal<string | null>(null);

  loading = signal(false);
  loadingMore = signal(false);
  selectedTabIndex = signal(0);
  showScrollTop = signal(false);

  private refreshTimer?: any;

  displayedWorkflowColumns = computed(() =>
    this.isHandset()
      ? ['enabled', 'name', 'actions']
      : ['enabled', 'name', 'description', 'steps', 'actions'],
  );
  displayedInstanceColumns = computed(() =>
    this.isHandset()
      ? ['workflowName', 'status', 'actions']
      : ['workflowName', 'trigger', 'status', 'startedAt', 'actions'],
  );

  fabConfig = computed(() => {
    if (this.selectedTabIndex() === 0) {
      return {
        icon: 'add',
        label: '创建工作流',
        action: () => this.createWorkflow(),
      };
    }
    return null;
  });

  constructor() {
    this.route.queryParams.subscribe((params) => {
      if (params['tab'] === 'instance') {
        this.selectedTabIndex.set(1);
        this.startRefreshTimer();
      } else {
        this.selectedTabIndex.set(0);
        this.stopRefreshTimer();
      }
      if (params['workflowId']) {
        this.selectedWorkflowId.set(params['workflowId']);
      } else {
        this.selectedWorkflowId.set(null);
      }
    });
  }

  @HostListener('window:scroll', [])
  onWindowScroll() {
    this.showScrollTop.set(window.scrollY > 300);
  }

  ngOnInit(): void {
    this.uiService.configureToolbar({ shadow: false });
    this.refreshAll();
    this.setupScrollListener();

    // Listen for search changes
    effect(() => {
      const search = this.uiService.searchConfig()?.value;
      this.refreshAll();
    });
  }

  ngOnDestroy(): void {
    this.uiService.resetToolbar();
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
        if (this.selectedTabIndex() === 0 && this.wfHasMore()) {
          this.loadWorkflows(false);
        } else if (this.selectedTabIndex() === 1 && this.instHasMore()) {
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
      if (this.selectedTabIndex() === 1 && !this.loading()) {
        this.loadInstances(true);
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
    const tab = index === 0 ? 'workflow' : 'instance';
    this.router.navigate([], {
      relativeTo: this.route,
      queryParams: { tab },
      queryParamsHandling: 'merge',
      replaceUrl: true,
    });

    if (index === 1) {
      this.startRefreshTimer();
    } else {
      this.stopRefreshTimer();
    }
    this.refreshData();
  }

  async refreshAll() {
    this.loading.set(true);
    try {
      await Promise.all([this.loadWorkflows(true), this.loadInstances(true, true)]);
    } catch (err) {
      this.snackBar
        .open('加载失败', '重试')
        .onAction()
        .subscribe(() => this.refreshAll());
    } finally {
      this.loading.set(false);
    }
  }

  async refreshData() {
    this.loading.set(true);
    try {
      if (this.selectedTabIndex() === 0) await this.loadWorkflows(true);
      else await this.loadInstances(true);
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
          20,
          this.uiService.searchConfig()?.value,
        ),
      );
      if (reset) {
        this.workflows.set(res.items || []);
      } else {
        const current = this.workflows();
        const newItems = (res.items || []).filter((n) => !current.some((e) => e.id === n.id));
        this.workflows.update((prev) => [...prev, ...newItems]);
      }
      this.wfTotal.set(res.total || 0);
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
          20,
          this.uiService.searchConfig()?.value,
        ),
      );

      let newItems = res.items || [];
      if (reset) {
        this.instances.set(newItems);
      } else {
        const current = this.instances();
        newItems = newItems.filter((n) => !current.some((e) => e.id === n.id));
        this.instances.update((prev) => [...prev, ...newItems]);
      }

      this.instTotal.set(res.total || 0);
      this.instNextCursor.set(res.nextCursor || '');
      this.instHasMore.set(res.hasMore || false);
    } finally {
      this.loadingMore.set(false);
      if (!silent) this.loading.set(false);
    }
  }

  getWorkflowName(id: string | undefined): string {
    if (!id) return '-';
    return this.workflows().find((w) => w.id === id)?.name || id;
  }

  isWorkflowRunning(workflowId: string | undefined): boolean {
    if (!workflowId) return false;
    return this.instances().some(
      (i) => i.workflowId === workflowId && (i.status === 'Running' || i.status === 'Pending'),
    );
  }

  getTriggerLabel(trigger: string | undefined): string {
    if (!trigger) return '-';
    if (trigger === 'Manual') return '手动执行';
    if (trigger === 'Webhook') return 'Webhook';
    if (trigger === 'Cron') return '定时计划';
    if (trigger === 'API') return 'API 调用';
    if (trigger.startsWith('SubWorkflow:')) return '子任务调用';
    return trigger;
  }

  getTriggerIcon(trigger: string | undefined): string {
    if (!trigger) return 'help_outline';
    if (trigger === 'Manual') return 'person';
    if (trigger === 'Webhook') return 'link';
    if (trigger === 'Cron') return 'schedule';
    if (trigger === 'API') return 'code';
    if (trigger.startsWith('SubWorkflow:')) return 'account_tree';
    return 'extension';
  }

  getTriggerTooltip(trigger: string | undefined): string {
    if (!trigger) return '';
    if (trigger.startsWith('SubWorkflow:')) {
      return '父级实例 ID: ' + trigger.split(':')[1];
    }
    return this.getTriggerLabel(trigger);
  }

  createWorkflow() {
    requestAnimationFrame(() => {
      const dialogRef = this.dialog.open(CreateWorkflowDialogComponent, {
        data: { workflow: null },
        width: '100vw',
        maxWidth: '100vw',
        height: '100vh',
        panelClass: 'full-screen-dialog',
      });

      dialogRef.afterClosed().subscribe(async (result) => {
        if (result) {
          this.loading.set(true);
          try {
            await firstValueFrom(this.orchService.actionsWorkflowsPost(result));
            this.snackBar.open('工作流已创建', '关闭', { duration: 2000 });
            await this.loadWorkflows();
          } catch (err: any) {
            this.snackBar.open('创建失败: ' + (err.error?.message || '未知错误'), '关闭');
          } finally {
            this.loading.set(false);
          }
        }
      });
    });
  }

  editWorkflow(wf: ModelsWorkflow) {
    requestAnimationFrame(() => {
      const dialogRef = this.dialog.open(CreateWorkflowDialogComponent, {
        data: { workflow: wf },
        width: '100vw',
        maxWidth: '100vw',
        height: '100vh',
        panelClass: 'full-screen-dialog',
      });

      dialogRef.afterClosed().subscribe(async (result) => {
        if (result && wf.id) {
          this.loading.set(true);
          try {
            await firstValueFrom(this.orchService.actionsWorkflowsIdPut(wf.id, result));
            this.snackBar.open('工作流已更新', '关闭', { duration: 2000 });
            await this.loadWorkflows();
          } catch (err: any) {
            this.snackBar.open('更新失败: ' + (err.error?.message || '未知错误'), '关闭');
          } finally {
            this.loading.set(false);
          }
        }
      });
    });
  }

  async deleteWorkflow(wf: ModelsWorkflow) {
    if (!wf.id) return;
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      data: {
        title: '删除工作流',
        message: `确定要删除工作流 "${wf.name}" 吗？`,
        confirmText: '确定删除',
        color: 'warn',
      },
    });

    dialogRef.afterClosed().subscribe(async (result) => {
      if (result && wf.id) {
        this.loading.set(true);
        try {
          await firstValueFrom(this.orchService.actionsWorkflowsIdDelete(wf.id));
          this.snackBar.open('工作流已删除', '关闭', { duration: 2000 });
          await this.loadWorkflows();
        } catch (err: any) {
          const msg = err.error?.message || err.message || '';
          if (msg.includes('permission denied') && msg.includes('write access required')) {
            this.snackBar.open('删除失败: 您没有该工作流的修改/删除权限', '了解', {
              duration: 5000,
            });
          } else {
            this.snackBar.open('删除失败', '关闭', { duration: 2000 });
          }
        } finally {
          this.loading.set(false);
        }
      }
    });
  }

  async runWorkflow(wf: ModelsWorkflow) {
    if (!wf.id) return;

    if (wf.vars && Object.keys(wf.vars).length > 0) {
      const dialogRef = this.dialog.open(RunWorkflowDialogComponent, {
        data: { workflow: wf },
        width: '400px',
      });

      dialogRef.afterClosed().subscribe(async (inputs) => {
        if (inputs) {
          this.executeRun(wf, inputs);
        }
      });
    } else {
      const dialogRef = this.dialog.open(ConfirmDialogComponent, {
        data: {
          title: '启动工作流',
          message: `确定要启动工作流 "${wf.name}" 吗？`,
          confirmText: '立即启动',
          color: 'primary',
        },
      });

      dialogRef.afterClosed().subscribe(async (result) => {
        if (result) {
          this.executeRun(wf, {});
        }
      });
    }
  }

  private async executeRun(wf: ModelsWorkflow, inputs: { [key: string]: string }) {
    this.loading.set(true);
    try {
      await firstValueFrom(this.orchService.actionsWorkflowsWorkflowIdRunPost(wf.id!, { inputs }));
      this.snackBar.open('工作流已启动', '关闭', { duration: 3000 });
      this.filterByWorkflow(wf.id!);
      await this.loadInstances();
    } catch (err: any) {
      const msg = err.error?.message || err.message || '';
      if (msg.includes('permission denied') && msg.includes('execution access required')) {
        this.snackBar.open('启动失败: 您没有该工作流的执行权限', '了解', { duration: 5000 });
      } else {
        this.snackBar.open('启动失败', '关闭', { duration: 2000 });
      }
    } finally {
      this.loading.set(false);
    }
  }

  async cancelInstance(inst: ModelsTaskInstance) {
    if (!inst.id) return;
    this.loading.set(true);
    try {
      await firstValueFrom(this.orchService.actionsInstancesIdCancelPost(inst.id));
      this.snackBar.open('任务已取消', '关闭', { duration: 2000 });
      await this.loadInstances();
    } catch (err) {
      this.snackBar.open('取消失败', '关闭', { duration: 2000 });
    } finally {
      this.loading.set(false);
    }
  }

  async deleteInstance(inst: ModelsTaskInstance) {
    if (!inst.id) return;
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      data: {
        title: '删除运行记录',
        message: `确定要删除此条运行记录吗？相关的执行日志也将被永久清除。`,
        confirmText: '确定删除',
        color: 'warn',
      },
    });

    dialogRef.afterClosed().subscribe(async (result) => {
      if (result && inst.id) {
        this.loading.set(true);
        try {
          await firstValueFrom(this.orchService.actionsInstancesIdDelete(inst.id));
          this.snackBar.open('记录已删除', '关闭', { duration: 2000 });
          await this.loadInstances();
        } catch (err) {
          this.snackBar.open('删除失败', '关闭', { duration: 2000 });
        } finally {
          this.loading.set(false);
        }
      }
    });
  }

  async cleanupInstances() {
    const days = 7; // Default to 7 days
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      data: {
        title: '批量清理记录',
        message: `确定要清理 ${days} 天前的所有运行记录吗？`,
        confirmText: '开始清理',
        color: 'warn',
      },
    });

    dialogRef.afterClosed().subscribe(async (result) => {
      if (result) {
        this.loading.set(true);
        try {
          const res = await firstValueFrom(this.orchService.actionsInstancesCleanupPost(days));
          this.snackBar.open(`清理完成，共删除 ${(res as any).deleted || 0} 条记录`, '关闭', {
            duration: 3000,
          });
          await this.loadInstances();
        } catch (err) {
          this.snackBar.open('清理失败', '关闭', { duration: 2000 });
        } finally {
          this.loading.set(false);
        }
      }
    });
  }

  filterByWorkflow(id: string | null) {
    const queryParams: any = { workflowId: id || null };
    if (id) {
      queryParams.tab = 'instance';
    }
    this.router.navigate([], {
      relativeTo: this.route,
      queryParams,
      queryParamsHandling: 'merge',
    });
  }

  viewLogs(inst: ModelsTaskInstance) {
    requestAnimationFrame(() => {
      this.dialog
        .open(TaskDetailDialogComponent, {
          data: { instance: inst },
          width: '100vw',
          maxWidth: '100vw',
          height: '100vh',
          panelClass: 'full-screen-dialog',
        })
        .afterClosed()
        .subscribe(() => {
          this.loadInstances();
        });
    });
  }

  async resetWebhookToken(wf: ModelsWorkflow) {
    if (!wf.id) return;
    this.loading.set(true);
    try {
      const newToken = await firstValueFrom(
        this.orchService.actionsWorkflowsIdWebhookResetPost(wf.id),
      );
      wf.webhookToken = newToken; // Update local ref
      this.snackBar
        .open('Webhook Token 已重置', '复制新地址', { duration: 5000 })
        .onAction()
        .subscribe(() => this.copyWebhookUrl(wf));
    } catch (err) {
      this.snackBar.open('重置失败', '关闭', { duration: 2000 });
    } finally {
      this.loading.set(false);
    }
  }

  copyWebhookUrl(wf: ModelsWorkflow) {
    if (!wf.webhookToken) return;
    const url = `${window.location.protocol}//${window.location.host}/api/v1/actions/webhooks/${wf.webhookToken}`;
    this.clipboard.copy(url);
    this.snackBar.open('Webhook URL 已复制到剪贴板', '确定', { duration: 2000 });
  }

  async toggleWorkflow(wf: ModelsWorkflow) {
    if (!wf.id) return;
    const originalStatus = wf.enabled;
    const newStatus = !originalStatus;

    // Optimistic update
    wf.enabled = newStatus;

    try {
      await firstValueFrom(this.orchService.actionsWorkflowsIdPut(wf.id, wf));
      this.snackBar.open(newStatus ? '工作流已启用' : '工作流已禁用', '关闭', { duration: 2000 });
    } catch (err) {
      wf.enabled = originalStatus; // Rollback
      this.snackBar.open('状态更新失败', '关闭', { duration: 2000 });
    }
  }

  getStatusClass(status: string | undefined): string {
    switch (status) {
      case 'Success':
        return 'bg-success/10 text-success border-success/20';
      case 'Failed':
        return 'bg-error/10 text-error border-error/20';
      case 'Running':
      case 'Pending':
        return 'bg-primary/10 text-primary border-primary/20';
      case 'Cancelled':
        return 'bg-surface-container-high text-outline border-outline-variant/30';
      default:
        return 'bg-surface-container text-on-surface border-outline-variant/30';
    }
  }

  openSearch() {
    this.uiService.openSearch({
      placeholder:
        this.selectedTabIndex() === 0 ? '搜索工作流名称或描述...' : '搜索实例 ID 或工作流...',
      value: this.uiService.searchConfig()?.value || '',
      onSearch: (val) => {
        const config = this.uiService.searchConfig();
        if (config) {
          this.uiService.searchConfig.set({ ...config, value: val });
        }
      },
    });
  }
}
