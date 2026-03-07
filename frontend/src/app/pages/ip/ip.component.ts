import { Component, OnInit, OnDestroy, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatTabsModule } from '@angular/material/tabs';
import { MatTableModule } from '@angular/material/table';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatCardModule } from '@angular/material/card';
import { MatDialog, MatDialogModule } from '@angular/material/dialog';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatMenuModule } from '@angular/material/menu';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { MatDividerModule } from '@angular/material/divider';
import { ActivatedRoute, Router } from '@angular/router';
import { BreakpointObserver, Breakpoints } from '@angular/cdk/layout';
import { toSignal } from '@angular/core/rxjs-interop';
import { map } from 'rxjs/operators';
import { FormsModule } from '@angular/forms';
import { firstValueFrom } from 'rxjs';

import { PageHeaderComponent } from '../../shared/page-header.component';
import { ConfirmDialogComponent } from '../rbac/confirm-dialog.component';
import { CreatePoolDialogComponent } from './create-pool-dialog.component';
import { ManageEntriesDialogComponent } from './manage-entries-dialog.component';
import { CreateExportDialogComponent } from './create-export-dialog.component';
import { ExportTasksDialogComponent } from '../../shared/export-tasks-dialog.component';
import { PreviewExportDialogComponent } from '../../shared/preview-export-dialog.component';
import { UiService } from '../../ui.service';
import { NetworkIpService, ModelsIPGroup, ModelsIPExport } from '../../generated';

@Component({
  selector: 'app-ip',
  standalone: true,
  imports: [
    CommonModule,
    MatTabsModule,
    MatTableModule,
    MatButtonModule,
    MatIconModule,
    MatCardModule,
    MatDialogModule,
    MatProgressBarModule,
    MatProgressSpinnerModule,
    MatTooltipModule,
    MatMenuModule,
    MatSlideToggleModule,
    MatDividerModule,
    PageHeaderComponent,
    FormsModule,
  ],
  templateUrl: './ip.component.html',
  styles: [
    `
      :host {
        display: block;
      }
      ::ng-deep .ip-tabs-integrated {
        .mat-mdc-tab-header {
          background: var(--mat-sys-surface);
          border-bottom: 1px solid var(--mat-sys-outline-variant);
        }
        .mat-mdc-tab-body-wrapper {
          background: var(--mat-sys-surface-container-lowest);
        }
      }
    `,
  ],
})
export class IpComponent implements OnInit, OnDestroy {
  private ipService = inject(NetworkIpService);
  private snackBar = inject(MatSnackBar);
  private dialog = inject(MatDialog);
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private breakpointObserver = inject(BreakpointObserver);
  public uiService = inject(UiService);

  private scrollListener?: () => void;

  isHandset = toSignal(
    this.breakpointObserver.observe(Breakpoints.Handset).pipe(map((result) => result.matches)),
    { initialValue: this.breakpointObserver.isMatched(Breakpoints.Handset) },
  );

  loading = signal(false);
  loadingMore = signal(false);
  selectedTabIndex = signal(0);
  showScrollTop = signal(false);

  // Address Pools
  pools = signal<ModelsIPGroup[]>([]);
  poolSearch = signal('');
  poolPage = signal(1);
  poolTotal = signal(0);

  // Dynamic Exports
  exports = signal<ModelsIPExport[]>([]);
  exportSearch = signal('');
  exportPage = signal(1);
  exportTotal = signal(0);

  activeTasks = signal<Record<string, any>>({});

  displayedPoolColumns = computed(() =>
    this.isHandset()
      ? ['name', 'entryCount', 'actions']
      : ['name', 'description', 'entryCount', 'updatedAt', 'actions'],
  );

  displayedExportColumns = computed(() =>
    this.isHandset() ? ['name', 'actions'] : ['name', 'rule', 'updatedAt', 'actions'],
  );

  hasSearchContent = computed(() => {
    return this.selectedTabIndex() === 0
      ? this.poolSearch().length > 0
      : this.exportSearch().length > 0;
  });

  fabConfig = computed(() => {
    if (this.selectedTabIndex() === 0) {
      return { icon: 'add', label: '新建地址池', action: () => this.createPool() };
    } else {
      return { icon: 'add', label: '新建导出配置', action: () => this.createExport() };
    }
  });

  ngOnInit() {
    this.uiService.configureToolbar({ shadow: false });
    this.route.queryParams.subscribe((params) => {
      if (params['tab'] === 'pool') this.selectedTabIndex.set(0);
      else if (params['tab'] === 'export') this.selectedTabIndex.set(1);

      if (params['search']) {
        if (this.selectedTabIndex() === 0) {
          this.poolSearch.set(params['search']);
        } else {
          this.exportSearch.set(params['search']);
        }
      }
      this.loadData(true);
    });
    this.setupScrollListener();
  }

  ngOnDestroy() {
    this.uiService.resetToolbar();
    this.uiService.closeSearch();
    if (this.scrollListener) {
      const scrollElement = document.querySelector('mat-sidenav-content');
      scrollElement?.removeEventListener('scroll', this.scrollListener);
    }
  }

  private setupScrollListener() {
    const scrollElement = document.querySelector('mat-sidenav-content');
    if (!scrollElement) return;

    this.scrollListener = () => {
      this.showScrollTop.set(scrollElement.scrollTop > 300);
      const atBottom =
        scrollElement.scrollHeight - scrollElement.scrollTop <= scrollElement.clientHeight + 150;

      if (atBottom && !this.loadingMore() && !this.loading()) {
        if (this.selectedTabIndex() === 0 && this.pools().length < this.poolTotal()) {
          this.poolPage.update((p) => p + 1);
          this.loadPools(false);
        } else if (this.selectedTabIndex() === 1 && this.exports().length < this.exportTotal()) {
          this.exportPage.update((p) => p + 1);
          this.loadExports(false);
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

  onTabChange(index: number) {
    this.selectedTabIndex.set(index);
    const tab = index === 0 ? 'pool' : 'export';
    this.router.navigate([], {
      relativeTo: this.route,
      queryParams: { tab },
      queryParamsHandling: 'merge',
    });
    this.loadData(true);
  }

  loadData(reset = false) {
    if (this.selectedTabIndex() === 0) this.loadPools(reset);
    else this.loadExports(reset);
  }

  async loadPools(reset = false) {
    if (reset) {
      this.loading.set(true);
      this.poolPage.set(1);
    } else {
      this.loadingMore.set(true);
    }

    try {
      const res = await firstValueFrom(
        this.ipService.networkIpPoolsGet(this.poolPage(), 20, this.poolSearch()),
      );
      if (reset) {
        this.pools.set(res.items || []);
      } else {
        const current = this.pools();
        const newItems = (res.items || []).filter((n) => !current.some((e) => e.id === n.id));
        this.pools.update((prev) => [...prev, ...newItems]);
      }
      this.poolTotal.set(res.total || 0);
    } catch (err) {
      console.error(err);
    } finally {
      this.loading.set(false);
      this.loadingMore.set(false);
    }
  }

  async loadExports(reset = false) {
    if (reset) {
      this.loading.set(true);
      this.exportPage.set(1);
    } else {
      this.loadingMore.set(true);
    }

    try {
      const res = await firstValueFrom(
        this.ipService.networkIpExportsGet(this.exportPage(), 20, this.exportSearch()),
      );
      if (reset) {
        this.exports.set(res.items || []);
      } else {
        const current = this.exports();
        const newItems = (res.items || []).filter((n) => !current.some((e) => e.id === n.id));
        this.exports.update((prev) => [...prev, ...newItems]);
      }
      this.exportTotal.set(res.total || 0);
    } catch (err) {
      console.error(err);
    } finally {
      this.loading.set(false);
      this.loadingMore.set(false);
    }
  }

  openSearch() {
    const isPool = this.selectedTabIndex() === 0;
    this.uiService.openSearch({
      placeholder: isPool ? '搜索地址池名称...' : '搜索导出配置名称...',
      value: isPool ? this.poolSearch() : this.exportSearch(),
      onSearch: (val) => {
        const queryParams: any = { search: val || null };
        this.router.navigate([], {
          relativeTo: this.route,
          queryParams,
          queryParamsHandling: 'merge',
        });

        if (isPool) {
          this.poolSearch.set(val);
        } else {
          this.exportSearch.set(val);
        }
        this.loadData(true);
      },
    });
  }

  createPool() {
    requestAnimationFrame(() => {
      const dialogRef = this.dialog.open(CreatePoolDialogComponent, { width: '400px', data: {} });
      dialogRef.afterClosed().subscribe((res) => {
        if (res) this.loadPools(true);
      });
    });
  }

  editPool(pool: ModelsIPGroup) {
    requestAnimationFrame(() => {
      const dialogRef = this.dialog.open(CreatePoolDialogComponent, {
        width: '400px',
        data: { pool },
      });
      dialogRef.afterClosed().subscribe((res) => {
        if (res) this.loadPools(true);
      });
    });
  }

  deletePool(pool: ModelsIPGroup) {
    requestAnimationFrame(() => {
      const dialogRef = this.dialog.open(ConfirmDialogComponent, {
        data: {
          title: '删除地址池',
          message: `确定要删除地址池 [${pool.name}] 吗？此操作将永久删除所有数据文件。`,
          color: 'warn',
        },
      });
      dialogRef.afterClosed().subscribe(async (res) => {
        if (res && pool.id) {
          try {
            await firstValueFrom(this.ipService.networkIpPoolsIdDelete(pool.id));
            this.snackBar.open('删除成功', '关闭', { duration: 3000 });
            this.loadPools(true);
          } catch (err: any) {
            this.snackBar.open(`删除失败: ${err.error?.message || err.message}`, '关闭', {
              duration: 3000,
            });
          }
        }
      });
    });
  }

  manageEntries(pool: ModelsIPGroup) {
    requestAnimationFrame(() => {
      this.dialog.open(ManageEntriesDialogComponent, {
        width: '100vw',
        height: '100vh',
        maxWidth: '100vw',
        maxHeight: '100vh',
        panelClass: 'full-screen-dialog',
        data: { pool },
      });
    });
  }

  createExport() {
    requestAnimationFrame(() => {
      const dialogRef = this.dialog.open(CreateExportDialogComponent, { width: '500px', data: {} });
      dialogRef.afterClosed().subscribe((res) => {
        if (res) this.loadExports(true);
      });
    });
  }

  editExport(exp: ModelsIPExport) {
    requestAnimationFrame(() => {
      const dialogRef = this.dialog.open(CreateExportDialogComponent, {
        width: '500px',
        data: { export: exp },
      });
      dialogRef.afterClosed().subscribe((res) => {
        if (res) this.loadExports(true);
      });
    });
  }

  deleteExport(exp: ModelsIPExport) {
    requestAnimationFrame(() => {
      const dialogRef = this.dialog.open(ConfirmDialogComponent, {
        data: {
          title: '删除配置',
          message: `确定要删除导出配置 [${exp.name}] 吗？此操作将级联删除该配置下的所有历史导出任务。`,
          color: 'warn',
        },
      });
      dialogRef.afterClosed().subscribe(async (res) => {
        if (res && exp.id) {
          try {
            await firstValueFrom(this.ipService.networkIpExportsIdDelete(exp.id));
            this.snackBar.open('删除成功', '关闭', { duration: 3000 });
            this.loadExports(true);
          } catch (err: any) {
            this.snackBar.open(`删除失败: ${err.error?.message || err.message}`, '关闭', {
              duration: 3000,
            });
          }
        }
      });
    });
  }

  async triggerExport(exp: ModelsIPExport, format: string = 'text') {
    if (!exp.id) return;
    try {
      await firstValueFrom(this.ipService.networkIpExportsIdTriggerPost(exp.id, format));
      this.openTasks();
    } catch (err: any) {
      this.snackBar.open(`触发失败: ${err.error?.message || err.message}`, '关闭', {
        duration: 3000,
      });
    }
  }

  openTasks() {
    this.dialog.open(ExportTasksDialogComponent, {
      width: '600px',
      data: { type: 'ip' },
      panelClass: 'tasks-dialog',
    });
  }

  previewExport() {
    this.dialog.open(PreviewExportDialogComponent, {
      width: '900px',
      maxWidth: '95vw',
      data: { type: 'ip', rule: '', groupIds: [] },
    });
  }
}
