import {
  Component,
  OnInit,
  OnDestroy,
  inject,
  signal,
  computed,
  HostListener,
} from '@angular/core';
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
import {
  NetworkSiteService,
  ModelsSiteGroup,
  ModelsSiteExport,
  ModelsSiteAnalysisResult,
} from '../../generated';
import { FormsModule } from '@angular/forms';
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';

@Component({
  selector: 'app-site',
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
    PageHeaderComponent,
    FormsModule,
    MatInputModule,
    MatFormFieldModule,
  ],
  templateUrl: './site.component.html',
  styles: [
    `
      :host {
        display: block;
      }
      .site-tabs-integrated {
        ::ng-deep .mat-mdc-tab-header {
          background: var(--mat-sys-surface);
          border-bottom: 1px solid var(--mat-sys-outline-variant);
          position: sticky;
          top: 0;
          z-index: 10;
        }
        ::ng-deep .mat-mdc-tab-body-wrapper {
          background: var(--mat-sys-surface-container-lowest);
        }
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

  selectedTabIndex = signal(0);
  loading = signal(false);
  showScrollTop = signal(false);

  // Pools state
  pools = signal<ModelsSiteGroup[]>([]);
  poolTotal = signal(0);
  poolPage = signal(1);
  poolSearch = signal('');

  // Exports state
  exports = signal<ModelsSiteExport[]>([]);
  exportTotal = signal(0);
  exportPage = signal(1);
  exportSearch = signal('');

  // Analysis state
  analysisDomain = signal('');
  analysisResult = signal<ModelsSiteAnalysisResult | null>(null);

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

  fabConfig = computed(() => {
    const tab = this.selectedTabIndex();
    if (tab === 0) return { icon: 'add', label: '新建域名池', action: () => this.createPool() };
    if (tab === 1) return { icon: 'add', label: '新建导出', action: () => this.createExport() };
    return null;
  });

  @HostListener('window:scroll', [])
  onWindowScroll() {
    this.showScrollTop.set(window.scrollY > 300);
  }

  ngOnInit() {
    this.route.queryParams.subscribe((params) => {
      if (params['tab'] === 'pool') this.selectedTabIndex.set(0);
      else if (params['tab'] === 'export') this.selectedTabIndex.set(1);
      else if (params['tab'] === 'analysis') this.selectedTabIndex.set(2);
      this.loadData();
    });
  }

  ngOnDestroy() {
    this.uiService.closeSearch();
  }

  scrollToTop() {
    window.scrollTo({ top: 0, behavior: 'smooth' });
  }

  onTabChange(index: number) {
    this.selectedTabIndex.set(index);
    let tab = 'pool';
    if (index === 1) tab = 'export';
    else if (index === 2) tab = 'analysis';
    this.router.navigate([], { queryParams: { tab }, queryParamsHandling: 'merge' });
    this.uiService.closeSearch();
  }

  openSearch() {
    const tab = this.selectedTabIndex();
    this.uiService.openSearch({
      placeholder: tab === 0 ? '搜索域名池名称...' : '搜索导出配置名称...',
      value: tab === 0 ? this.poolSearch() : this.exportSearch(),
      onSearch: (val) => {
        if (tab === 0) this.onPoolSearch(val);
        else if (tab === 1) this.onExportSearch(val);
      },
    });
  }

  onPoolSearch(val: string) {
    this.poolSearch.set(val);
    this.poolPage.set(1);
    this.loadPools();
  }

  onExportSearch(val: string) {
    this.exportSearch.set(val);
    this.exportPage.set(1);
    this.loadExports();
  }

  loadData() {
    if (this.selectedTabIndex() === 0) {
      this.loadPools();
    } else if (this.selectedTabIndex() === 1) {
      this.loadExports();
    }
  }

  loadPools(reset = false) {
    if (reset) this.poolPage.set(1);
    this.loading.set(true);
    this.siteService.networkSitePoolsGet(this.poolPage(), 20, this.poolSearch()).subscribe({
      next: (res) => {
        this.pools.set(res.items || []);
        this.poolTotal.set(res.total || 0);
        this.loading.set(false);
      },
      error: () => this.loading.set(false),
    });
  }

  loadExports(reset = false) {
    if (reset) this.exportPage.set(1);
    this.loading.set(true);
    this.siteService.networkSiteExportsGet(this.exportPage(), 20, this.exportSearch()).subscribe({
      next: (res) => {
        this.exports.set(res.items || []);
        this.exportTotal.set(res.total || 0);
        this.loading.set(false);
      },
      error: () => this.loading.set(false),
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
      data: { type: 'site', rule: '', groupIds: [] }
    });
  }

  runAnalysis() {
    if (!this.analysisDomain()) return;
    this.loading.set(true);
    this.siteService
      .networkSiteAnalysisHitTestPost({ domain: this.analysisDomain(), groupIds: [] })
      .subscribe({
        next: (res) => {
          this.analysisResult.set(res);
          this.loading.set(false);
        },
        error: (err) => {
          this.snackBar.open(`分析失败: ${err.error?.message || err.message}`, '关闭', {
            duration: 3000,
          });
          this.loading.set(false);
        },
      });
  }
}
