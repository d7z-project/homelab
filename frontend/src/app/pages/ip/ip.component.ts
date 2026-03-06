import { Component, OnInit, OnDestroy, inject, signal, computed, HostListener } from '@angular/core';
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
import { ActivatedRoute, Router } from '@angular/router';
import { BreakpointObserver, Breakpoints } from '@angular/cdk/layout';
import { toSignal } from '@angular/core/rxjs-interop';
import { map } from 'rxjs/operators';

import { PageHeaderComponent } from '../../shared/page-header.component';
import { ConfirmDialogComponent } from '../rbac/confirm-dialog.component';
import { CreatePoolDialogComponent } from './create-pool-dialog.component';
import { CreateExportDialogComponent } from './create-export-dialog.component';
import { ManageEntriesDialogComponent } from './manage-entries-dialog.component';
import { UiService } from '../../ui.service';
import { NetworkIpService, ModelsIPGroup, ModelsIPExport, IpExportTask, ModelsIPAnalysisResult } from '../../generated';
import { FormsModule } from '@angular/forms';
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';

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
    PageHeaderComponent,
    FormsModule,
    MatInputModule,
    MatFormFieldModule,
  ],
  templateUrl: './ip.component.html',
  styles: [`
    :host { display: block; }
    .ip-tabs-integrated {
      ::ng-deep .mat-mdc-tab-header {
        background: var(--mat-sys-surface);
        border-bottom: 1px solid var(--mat-sys-outline-variant);
        position: sticky;
        top: 0;
        z-index: 10;
      }
      ::ng-deep .mat-mdc-tab-body-wrapper { background: var(--mat-sys-surface-container-lowest); }
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
  `]
})
export class IpComponent implements OnInit, OnDestroy {
  private ipService = inject(NetworkIpService);
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
  pools = signal<ModelsIPGroup[]>([]);
  poolTotal = signal(0);
  poolPage = signal(1);
  poolSearch = signal('');

  // Exports state
  exports = signal<ModelsIPExport[]>([]);
  exportTotal = signal(0);
  exportPage = signal(1);
  exportSearch = signal('');
  activeTasks = signal<Record<string, IpExportTask>>({});

  // Analysis state
  analysisIp = signal('');
  analysisResult = signal<ModelsIPAnalysisResult | null>(null);
  analysisInfo = signal<any | null>(null);

  displayedPoolColumns = computed(() =>
    this.isHandset() ? ['name', 'actions'] : ['name', 'description', 'entryCount', 'updatedAt', 'actions']
  );

  displayedExportColumns = computed(() =>
    this.isHandset() ? ['name', 'actions'] : ['name', 'rule', 'updatedAt', 'actions']
  );

  hasSearchContent = computed(() => {
    const tab = this.selectedTabIndex();
    if (tab === 0) return !!this.poolSearch();
    if (tab === 1) return !!this.exportSearch();
    return false;
  });

  fabConfig = computed(() => {
    const tab = this.selectedTabIndex();
    if (tab === 0) return { icon: 'add', label: '新建地址池', action: () => this.createPool() };
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
      placeholder: tab === 0 ? '搜索地址池名称...' : '搜索导出配置名称...',
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
    this.ipService.networkIpPoolsGet(this.poolPage(), 20, this.poolSearch()).subscribe({
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
    this.ipService.networkIpExportsGet(this.exportPage(), 20, this.exportSearch()).subscribe({
      next: (res) => {
        this.exports.set(res.items || []);
        this.exportTotal.set(res.total || 0);
        this.loading.set(false);
      },
      error: () => this.loading.set(false),
    });
  }

  createPool() {
    const dialogRef = this.dialog.open(CreatePoolDialogComponent, { width: '400px' });
    dialogRef.afterClosed().subscribe((res) => {
      if (res) this.loadPools(true);
    });
  }

  manageEntries(pool: ModelsIPGroup) {
    const dialogRef = this.dialog.open(ManageEntriesDialogComponent, {
      width: '100vw',
      height: '100vh',
      maxWidth: '100vw',
      panelClass: 'full-screen-dialog',
      data: { pool }
    });
    dialogRef.afterClosed().subscribe(() => {
      this.loadPools();
    });
  }

  deletePool(pool: ModelsIPGroup) {
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      data: { title: '删除确认', message: `确定要删除池 [${pool.name}] 吗？` },
    });
    dialogRef.afterClosed().subscribe((res) => {
      if (res && pool.id) {
        this.ipService.networkIpPoolsIdDelete(pool.id).subscribe({
          next: () => {
            this.snackBar.open('删除成功', '关闭', { duration: 3000 });
            this.loadPools();
          },
          error: (err) => {
            this.snackBar.open(`删除失败: ${err.error?.message || err.message}`, '关闭', { duration: 3000 });
          },
        });
      }
    });
  }

  createExport() {
    const dialogRef = this.dialog.open(CreateExportDialogComponent, { width: '600px' });
    dialogRef.afterClosed().subscribe((res) => {
      if (res) this.loadExports(true);
    });
  }

  deleteExport(exp: ModelsIPExport) {
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      data: { title: '删除确认', message: `确定要删除导出配置 [${exp.name}] 吗？` },
    });
    dialogRef.afterClosed().subscribe((res) => {
      if (res && exp.id) {
        this.ipService.networkIpExportsIdDelete(exp.id).subscribe({
          next: () => {
            this.snackBar.open('删除成功', '关闭', { duration: 3000 });
            this.loadExports();
          },
          error: (err) => {
            this.snackBar.open(`删除失败: ${err.error?.message || err.message}`, '关闭', { duration: 3000 });
          },
        });
      }
    });
  }

  triggerExport(exp: ModelsIPExport) {
    if (!exp.id) return;
    this.ipService.networkIpExportsIdTriggerPost(exp.id, 'text').subscribe({
      next: (res: any) => {
        const taskId = res.taskId;
        this.snackBar.open(`导出任务已触发`, '关闭', { duration: 2000 });
        this.pollTaskStatus(taskId, exp.id as string);
      },
      error: (err) => {
        this.snackBar.open(`触发失败: ${err.error?.message || err.message}`, '关闭', { duration: 3000 });
      }
    });
  }

  pollTaskStatus(taskId: string, exportId: string) {
    const interval = setInterval(() => {
      this.ipService.networkIpExportsTaskTaskIdGet(taskId).subscribe({
        next: (task) => {
          this.activeTasks.update(v => ({...v, [exportId]: task}));
          if (task.status === 'Success' || task.status === 'Failed' || task.status === 'Cancelled') {
            clearInterval(interval);
          }
        },
        error: () => clearInterval(interval)
      });
    }, 1000);
  }

  runAnalysis() {
    if (!this.analysisIp()) return;
    this.loading.set(true);
    this.analysisResult.set(null);
    this.analysisInfo.set(null);

    // Call HitTest
    this.ipService.networkIpAnalysisHitTestPost({ ip: this.analysisIp(), groupIds: [] }).subscribe({
      next: (res) => {
        this.analysisResult.set(res);
        if (!this.analysisInfo()) this.loading.set(false);
      },
      error: (err) => {
        this.snackBar.open(`分析失败: ${err.error?.message || err.message}`, '关闭', { duration: 3000 });
        this.loading.set(false);
      }
    });

    // Call Intelligence Info
    this.ipService.networkIpAnalysisInfoGet(this.analysisIp()).subscribe({
      next: (res) => {
        this.analysisInfo.set(res);
        if (this.analysisResult()) this.loading.set(false);
      },
      error: () => {
        // Silently fail or handle MMDB error
        if (this.analysisResult()) this.loading.set(false);
      }
    });
  }
}