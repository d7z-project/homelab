import {
  Component,
  OnInit,
  OnDestroy,
  inject,
  signal,
  computed,
  HostListener,
  untracked,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatTabsModule } from '@angular/material/tabs';
import { MatTableModule } from '@angular/material/table';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatDividerModule } from '@angular/material/divider';
import { MatMenuModule } from '@angular/material/menu';
import { MatDialog, MatDialogModule } from '@angular/material/dialog';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatTooltipModule } from '@angular/material/tooltip';
import { ActivatedRoute, Router } from '@angular/router';
import { BreakpointObserver, Breakpoints } from '@angular/cdk/layout';
import { toSignal } from '@angular/core/rxjs-interop';
import { map } from 'rxjs/operators';

import { PageHeaderComponent } from '../../shared/page-header.component';
import { ConfirmDialogComponent } from '../rbac/confirm-dialog.component';
import { CreateSitePoolDialogComponent } from './create-site-pool-dialog.component';
import { CreateSiteExportDialogComponent } from './create-site-export-dialog.component';
import { ManageSiteEntriesDialogComponent } from './manage-site-entries-dialog.component';
import { ExportTasksDialogComponent } from '../../shared/export-tasks-dialog.component';
import { PreviewExportDialogComponent } from '../../shared/preview-export-dialog.component';
import { UiService } from '../../ui.service';
import { NetworkSiteService, ModelsSiteGroup, ModelsSiteExport } from '../../generated';
import { FormsModule } from '@angular/forms';

@Component({
  selector: 'app-site',
  standalone: true,
  imports: [
    CommonModule,
    MatTabsModule,
    MatTableModule,
    MatButtonModule,
    MatIconModule,
    MatDividerModule,
    MatMenuModule,
    MatDialogModule,
    MatProgressBarModule,
    MatProgressSpinnerModule,
    MatTooltipModule,
    PageHeaderComponent,
    FormsModule,
  ],
  templateUrl: './site.component.html',
  styles: [
    `
      :host {
        display: block;
      }
      .search-field-m3 {
        ::ng-deep .mdc-text-field--filled {
          background-color: transparent !important;
        }
        ::ng-deep .mdc-line-ripple {
          display: none;
        }
        ::ng-deep .mat-mdc-form-field-subscript-wrapper {
          display: none;
        }
        ::ng-deep .mat-mdc-text-field-wrapper {
          padding-bottom: 0;
        }
      }
    `,
  ],
})
export class SiteComponent implements OnInit, OnDestroy {
  private siteService = inject(NetworkSiteService);
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

  // Pools state
  pools = signal<ModelsSiteGroup[]>([]);
  poolTotal = signal(0);
  poolNextCursor = signal('');
  poolSearch = signal('');
  hasMorePools = signal(false);

  // Exports state
  exports = signal<ModelsSiteExport[]>([]);
  exportTotal = signal(0);
  exportNextCursor = signal('');
  exportSearch = signal('');
  hasMoreExports = signal(false);

  displayedPoolColumns = computed(() =>
    this.isHandset()
      ? ['name', 'actions']
      : ['name', 'description', 'entryCount', 'updatedAt', 'actions'],
  );

  displayedExportColumns = computed(() =>
    this.isHandset() ? ['name', 'actions'] : ['name', 'rule', 'updatedAt', 'actions'],
  );

  hasSearchContent = computed(() => {
    const tab = this.selectedTabIndex();
    if (tab === 0) return !!this.poolSearch();
    if (tab === 1) return !!this.exportSearch();
    return false;
  });

  loading = signal(false);
  loadingMore = signal(false);
  selectedTabIndex = signal(0);
  showScrollTop = signal(false);

  fabConfig = computed(() => {
    const tab = this.selectedTabIndex();
    if (tab === 0) return { icon: 'add', label: '新建域名池', action: () => this.createPool() };
    if (tab === 1) return { icon: 'add', label: '新建导出', action: () => this.createExport() };
    return null;
  });

  constructor() {
    this.route.queryParams.subscribe((params) => {
      const tab = params['tab'];
      if (tab === 'export') {
        this.selectedTabIndex.set(1);
      } else {
        this.selectedTabIndex.set(0);
      }

      const search = params['search'] || '';
      if (this.selectedTabIndex() === 0) {
        if (search !== this.poolSearch()) {
          this.poolSearch.set(search);
          untracked(() => this.loadPools(true));
        }
      } else if (this.selectedTabIndex() === 1) {
        if (search !== this.exportSearch()) {
          this.exportSearch.set(search);
          untracked(() => this.loadExports(true));
        }
      }
    });
  }

  ngOnInit() {
    this.setupScrollListener();
    this.loadData();
  }

  ngOnDestroy() {
    this.uiService.closeSearch();
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
        if (tab === 0 && this.hasMorePools()) {
          this.loadPools(false);
        } else if (tab === 1 && this.hasMoreExports()) {
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
    let tab = 'pool';
    if (index === 1) tab = 'export';

    const search = index === 0 ? this.poolSearch() : index === 1 ? this.exportSearch() : '';

    this.router.navigate([], {
      relativeTo: this.route,
      queryParams: { tab, search: search || null },
      queryParamsHandling: 'merge',
    });
    this.uiService.closeSearch();
    this.loadData();
  }

  openSearch() {
    const tab = this.selectedTabIndex();
    this.uiService.openSearch({
      placeholder: tab === 0 ? '搜索域名池名称...' : '搜索导出配置名称...',
      value: tab === 0 ? this.poolSearch() : this.exportSearch(),
      onSearch: (val) => {
        if (tab === 0) {
          this.poolSearch.set(val);
        } else if (tab === 1) {
          this.exportSearch.set(val);
        }

        this.router.navigate([], {
          relativeTo: this.route,
          queryParams: { search: val || null },
          queryParamsHandling: 'merge',
        });

        this.loadData();
      },
    });
  }

  loadData() {
    if (this.selectedTabIndex() === 0) {
      this.loadPools(true);
    } else if (this.selectedTabIndex() === 1) {
      this.loadExports(true);
    }
  }

  loadPools(reset = false) {
    if (reset) {
      this.poolNextCursor.set('');
      this.loading.set(true);
    } else {
      this.loadingMore.set(true);
    }

    this.siteService.networkSitePoolsGet(this.poolNextCursor(), 20, this.poolSearch()).subscribe({
      next: (res) => {
        if (reset) {
          this.pools.set(res.items || []);
        } else {
          const current = this.pools();
          const newItems = (res.items || []).filter((n) => !current.some((e) => e.id === n.id));
          this.pools.update((prev) => [...prev, ...newItems]);
        }
        this.poolTotal.set(res.total || 0);
        this.poolNextCursor.set(res.nextCursor || '');
        this.hasMorePools.set(res.hasMore || false);
        this.loading.set(false);
        this.loadingMore.set(false);
      },
      error: () => {
        this.loading.set(false);
        this.loadingMore.set(false);
      },
    });
  }

  loadExports(reset = false) {
    if (reset) {
      this.exportNextCursor.set('');
      this.loading.set(true);
    } else {
      this.loadingMore.set(true);
    }

    this.siteService
      .networkSiteExportsGet(this.exportNextCursor(), 20, this.exportSearch())
      .subscribe({
        next: (res) => {
          if (reset) {
            this.exports.set(res.items || []);
          } else {
            const current = this.exports();
            const newItems = (res.items || []).filter((n) => !current.some((e) => e.id === n.id));
            this.exports.update((prev) => [...prev, ...newItems]);
          }
          this.exportTotal.set(res.total || 0);
          this.exportNextCursor.set(res.nextCursor || '');
          this.hasMoreExports.set(res.hasMore || false);
          this.loading.set(false);
          this.loadingMore.set(false);
        },
        error: () => {
          this.loading.set(false);
          this.loadingMore.set(false);
        },
      });
  }

  createPool() {
    const dialogRef = this.dialog.open(CreateSitePoolDialogComponent, { width: '400px' });
    dialogRef.afterClosed().subscribe((res) => {
      if (res) this.loadPools(true);
    });
  }

  manageEntries(pool: ModelsSiteGroup) {
    const dialogRef = this.dialog.open(ManageSiteEntriesDialogComponent, {
      width: '100vw',
      height: '100vh',
      maxWidth: '100vw',
      panelClass: 'full-screen-dialog',
      data: { pool },
    });
    dialogRef.afterClosed().subscribe(() => {
      this.loadPools();
    });
  }

  deletePool(pool: ModelsSiteGroup) {
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      data: { title: '删除确认', message: `确定要删除域名池 [${pool.name}] 吗？` },
    });
    dialogRef.afterClosed().subscribe((res) => {
      if (res && pool.id) {
        this.siteService.networkSitePoolsIdDelete(pool.id).subscribe({
          next: () => {
            this.snackBar.open('删除成功', '关闭', { duration: 3000 });
            this.loadPools();
          },
          error: (err) => {
            this.snackBar.open(`删除失败: ${err.error?.message || err.message}`, '关闭', {
              duration: 3000,
            });
          },
        });
      }
    });
  }

  createExport() {
    const dialogRef = this.dialog.open(CreateSiteExportDialogComponent, { width: '600px' });
    dialogRef.afterClosed().subscribe((res) => {
      if (res) this.loadExports(true);
    });
  }

  deleteExport(exp: ModelsSiteExport) {
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      data: {
        title: '删除确认',
        message: `确定要删除导出配置 [${exp.name}] 吗？此操作将级联删除该配置下的所有历史导出任务。`,
        color: 'warn',
      },
    });
    dialogRef.afterClosed().subscribe((res) => {
      if (res && exp.id) {
        this.siteService.networkSiteExportsIdDelete(exp.id).subscribe({
          next: () => {
            this.snackBar.open('删除成功', '关闭', { duration: 3000 });
            this.loadExports();
          },
          error: (err) => {
            this.snackBar.open(`删除失败: ${err.error?.message || err.message}`, '关闭', {
              duration: 3000,
            });
          },
        });
      }
    });
  }

  triggerExport(exp: ModelsSiteExport, format: string = 'text') {
    if (!exp.id) return;
    this.siteService.networkSiteExportsIdTriggerPost(exp.id, format).subscribe({
      next: (res: any) => {
        this.openTasks();
      },
      error: (err) => {
        this.snackBar.open(`触发失败: ${err.error?.message || err.message}`, '关闭', {
          duration: 3000,
        });
      },
    });
  }

  openTasks() {
    this.dialog.open(ExportTasksDialogComponent, {
      width: '600px',
      data: { type: 'site' },
      panelClass: 'tasks-dialog',
    });
  }

  previewExport() {
    this.dialog.open(PreviewExportDialogComponent, {
      width: '900px',
      maxWidth: '95vw',
      data: { type: 'site', rule: '', groupIds: [] },
    });
  }
}
