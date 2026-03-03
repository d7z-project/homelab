import { Component, OnInit, inject, signal, computed, OnDestroy, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { MatTableModule } from '@angular/material/table';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatCardModule } from '@angular/material/card';
import { MatTabsModule } from '@angular/material/tabs';
import { MatDividerModule } from '@angular/material/divider';
import { OrchestrationService, ModelsWorkflow, ModelsTaskInstance } from '../../generated';
import { MatDialog, MatDialogModule } from '@angular/material/dialog';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatMenuModule } from '@angular/material/menu';
import { MatChipsModule } from '@angular/material/chips';
import { firstValueFrom } from 'rxjs';
import { BreakpointObserver, Breakpoints } from '@angular/cdk/layout';
import { toSignal } from '@angular/core/rxjs-interop';
import { map } from 'rxjs/operators';
import { UiService } from '../../ui.service';
import { ConfirmDialogComponent } from '../rbac/confirm-dialog.component';
import { CreateWorkflowDialogComponent } from './create-workflow-dialog.component';
import { TaskDetailDialogComponent } from './task-detail-dialog.component';

@Component({
  selector: 'app-orchestration',
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
  ],
  templateUrl: './orchestration.component.html',
})
export class OrchestrationComponent implements OnInit, OnDestroy {
  private orchService = inject(OrchestrationService);
  private snackBar = inject(MatSnackBar);
  private dialog = inject(MatDialog);
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

  loading = signal(false);
  selectedTabIndex = signal(0);
  showScrollTop = signal(false);

  displayedWorkflowColumns = computed(() =>
    this.isHandset() ? ['name', 'actions'] : ['name', 'description', 'steps', 'actions'],
  );
  displayedInstanceColumns = computed(() =>
    this.isHandset() ? ['id', 'status', 'actions'] : ['id', 'workflowId', 'status', 'startedAt', 'actions'],
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
      if (params['tab'] === 'instance') this.selectedTabIndex.set(1);
      else this.selectedTabIndex.set(0);
    });
  }

  @HostListener('window:scroll', [])
  onWindowScroll() {
    this.showScrollTop.set(window.scrollY > 300);
  }

  scrollToTop() {
    window.scrollTo({ top: 0, behavior: 'smooth' });
  }

  ngOnInit(): void {
    this.uiService.configureToolbar({ shadow: false });
    this.refreshAll();
  }

  ngOnDestroy(): void {
    this.uiService.resetToolbar();
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
    this.refreshData();
  }

  async refreshAll() {
    this.loading.set(true);
    try {
      await Promise.all([this.loadWorkflows(), this.loadInstances()]);
    } catch (err) {
      this.snackBar.open('加载失败', '重试').onAction().subscribe(() => this.refreshAll());
    } finally {
      this.loading.set(false);
    }
  }

  async refreshData() {
    this.loading.set(true);
    try {
      if (this.selectedTabIndex() === 0) await this.loadWorkflows();
      else await this.loadInstances();
    } finally {
      this.loading.set(false);
    }
  }

  async loadWorkflows() {
    const data = await firstValueFrom(this.orchService.orchestrationWorkflowsGet());
    this.workflows.set(data || []);
  }

  async loadInstances() {
    const data = await firstValueFrom(this.orchService.orchestrationInstancesGet());
    // Sort by startedAt descending
    const sorted = (data || []).sort((a, b) => {
        return new Date(b.startedAt || 0).getTime() - new Date(a.startedAt || 0).getTime();
    });
    this.instances.set(sorted);
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
            await firstValueFrom(this.orchService.orchestrationWorkflowsPost(result));
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
            await firstValueFrom(this.orchService.orchestrationWorkflowsIdPut(wf.id, result));
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
          await firstValueFrom(this.orchService.orchestrationWorkflowsIdDelete(wf.id));
          this.snackBar.open('工作流已删除', '关闭', { duration: 2000 });
          await this.loadWorkflows();
        } catch (err) {
          this.snackBar.open('删除失败', '关闭', { duration: 2000 });
        } finally {
          this.loading.set(false);
        }
      }
    });
  }

  async runWorkflow(wf: ModelsWorkflow) {
    if (!wf.id) return;
    this.loading.set(true);
    try {
      await firstValueFrom(this.orchService.orchestrationWorkflowsWorkflowIdRunPost(wf.id));
      this.snackBar.open('工作流已启动', '查看实例', { duration: 5000 })
        .onAction().subscribe(() => this.onTabChange(1));
      await this.loadInstances();
    } catch (err) {
      this.snackBar.open('启动失败', '关闭', { duration: 2000 });
    } finally {
      this.loading.set(false);
    }
  }

  async cancelInstance(inst: ModelsTaskInstance) {
    if (!inst.id) return;
    this.loading.set(true);
    try {
      await firstValueFrom(this.orchService.orchestrationInstancesIdCancelPost(inst.id));
      this.snackBar.open('任务已取消', '关闭', { duration: 2000 });
      await this.loadInstances();
    } catch (err) {
      this.snackBar.open('取消失败', '关闭', { duration: 2000 });
    } finally {
      this.loading.set(false);
    }
  }

  viewLogs(inst: ModelsTaskInstance) {
    requestAnimationFrame(() => {
      this.dialog.open(TaskDetailDialogComponent, {
        data: { instance: inst },
        width: '100vw',
        maxWidth: '100vw',
        height: '100vh',
        panelClass: 'full-screen-dialog',
      }).afterClosed().subscribe(() => {
          this.loadInstances();
      });
    });
  }

  getStatusClass(status: string | undefined): string {
    switch (status) {
      case 'Success': return 'bg-success/10 text-success border-success/20';
      case 'Failed': return 'bg-error/10 text-error border-error/20';
      case 'Running': return 'bg-primary/10 text-primary border-primary/20';
      case 'Cancelled': return 'bg-surface-container-high text-outline border-outline-variant/30';
      default: return 'bg-surface-container text-on-surface border-outline-variant/30';
    }
  }

  openSearch() {
    this.uiService.openSearch({
      placeholder: this.selectedTabIndex() === 0 ? '搜索工作流名称或描述...' : '搜索实例 ID 或工作流...',
      value: '',
      onSearch: (val) => {
        // Implement local filtering or remote search if needed
        console.log('Searching for:', val);
      },
    });
  }
}
