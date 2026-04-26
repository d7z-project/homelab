import { Component, OnInit, OnDestroy, inject, signal, computed, untracked } from '@angular/core';
import { CommonModule } from '@angular/common';
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
import { MatDividerModule } from '@angular/material/divider';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { MatTabsModule } from '@angular/material/tabs';
import { BreakpointObserver, Breakpoints } from '@angular/cdk/layout';
import { toSignal } from '@angular/core/rxjs-interop';
import { map } from 'rxjs/operators';
import { FormsModule } from '@angular/forms';
import { firstValueFrom } from 'rxjs';
import { Router, ActivatedRoute } from '@angular/router';

import { PageHeaderComponent } from '../../shared/page-header.component';
import { ConfirmDialogComponent } from '../rbac/confirm-dialog.component';
import { CreateSyncPolicyDialogComponent } from './create-sync-policy-dialog.component';
import { UiService } from '../../ui.service';
import {
  NetworkSiteService,
  HomelabPkgApisNetworkSiteV1SyncPolicy,
  V1Group,
} from '../../generated';

@Component({
  selector: 'app-site-sync',
  standalone: true,
  imports: [
    CommonModule,
    MatTableModule,
    MatButtonModule,
    MatIconModule,
    MatCardModule,
    MatDialogModule,
    MatProgressBarModule,
    MatProgressSpinnerModule,
    MatTooltipModule,
    MatMenuModule,
    MatDividerModule,
    MatSlideToggleModule,
    MatTabsModule,
    PageHeaderComponent,
    FormsModule,
  ],
  templateUrl: './sync.component.html',
  styles: [
    `
      :host {
        display: block;
      }
      ::ng-deep .sync-tabs-integrated {
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
export class SiteSyncComponent implements OnInit, OnDestroy {
  private siteService = inject(NetworkSiteService);
  private snackBar = inject(MatSnackBar);
  private dialog = inject(MatDialog);
  private breakpointObserver = inject(BreakpointObserver);
  private router = inject(Router);
  private route = inject(ActivatedRoute);
  public uiService = inject(UiService);

  private scrollListener?: () => void;

  isHandset = toSignal(
    this.breakpointObserver.observe(Breakpoints.Handset).pipe(map((result) => result.matches)),
    { initialValue: this.breakpointObserver.isMatched(Breakpoints.Handset) },
  );

  loading = signal(false);
  loadingMore = signal(false);
  syncingRows = signal<Record<string, boolean>>({}); // Track per-row syncing
  policies = signal<HomelabPkgApisNetworkSiteV1SyncPolicy[]>([]);
  groups = signal<Map<string, string>>(new Map()); // ID -> Name
  search = signal('');
  nextCursor = signal('');
  pageSize = signal(20);
  total = signal(0);
  hasMore = signal(false);
  showScrollTop = signal(false);

  displayedColumns = computed(() =>
    this.isHandset()
      ? ['enabled', 'name', 'status', 'actions']
      : ['enabled', 'name', 'target', 'format', 'mode', 'cron', 'status', 'actions'],
  );

  hasSearchContent = computed(() => this.search().length > 0);

  // 是否有任何策略正在同步中
  anySyncing = computed(() =>
    this.policies().some(
      (p) => p.status?.lastStatus === 'Pending' || p.status?.lastStatus === 'Running',
    ),
  );

  private refreshTimer?: any;

  constructor() {
    this.route.queryParams.subscribe((params) => {
      const search = params['search'] || '';
      if (search !== this.search()) {
        this.search.set(search);
        untracked(() => this.loadPolicies(true));
      }
    });
  }

  ngOnInit() {
    this.loadGroups();
    this.loadPolicies(true);
    this.setupScrollListener();
    this.setupRefreshTimer();
  }

  ngOnDestroy() {
    this.uiService.closeSearch();
    if (this.scrollListener) {
      const scrollElement = document.querySelector('mat-sidenav-content');
      scrollElement?.removeEventListener('scroll', this.scrollListener);
    }
    this.stopRefreshTimer();
  }

  private setupRefreshTimer() {
    this.refreshTimer = setInterval(() => {
      // 如果有任何正在同步的任务，或者当前处于加载状态，则自动刷新
      if (this.anySyncing() && !this.loading()) {
        this.loadPolicies(true);
      }
    }, 3000); // 3秒刷新一次
  }

  private stopRefreshTimer() {
    if (this.refreshTimer) {
      clearInterval(this.refreshTimer);
    }
  }

  private setupScrollListener() {
    const scrollElement = document.querySelector('mat-sidenav-content');
    if (!scrollElement) return;

    this.scrollListener = () => {
      this.showScrollTop.set(scrollElement.scrollTop > 300);
      const atBottom =
        scrollElement.scrollHeight - scrollElement.scrollTop <= scrollElement.clientHeight + 150;

      if (atBottom && !this.loadingMore() && !this.loading() && this.hasMore()) {
        this.loadPolicies(false);
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

  loadGroups() {
    this.siteService.networkSitePoolsGet('', 100).subscribe({
      next: (res) => {
        const m = new Map<string, string>();
        (res.items || []).forEach((g) => m.set(g.id || '', g.meta?.name || ''));
        this.groups.set(m);
      },
    });
  }

  getGroupName(id?: string) {
    if (!id) return '未知池';
    return this.groups().get(id) || id;
  }

  goToPool(id?: string) {
    if (!id) return;
    const name = this.groups().get(id);
    this.router.navigate(['/network/site'], {
      queryParams: {
        tab: 'pool',
        search: name || id,
      },
    });
  }

  openSearch() {
    this.uiService.openSearch({
      placeholder: '搜索策略名称或 ID...',
      value: this.search(),
      onSearch: (val) => {
        this.search.set(val);
        this.router.navigate([], {
          relativeTo: this.route,
          queryParams: { search: val || null },
          queryParamsHandling: 'merge',
        });
        this.loadPolicies(true);
      },
    });
  }

  async loadPolicies(reset = false) {
    if (reset) {
      this.loading.set(true);
      this.nextCursor.set('');
    } else {
      this.loadingMore.set(true);
    }

    try {
      const res = await firstValueFrom(
        this.siteService.networkSiteSyncGet(this.nextCursor(), this.pageSize(), this.search()),
      );
      const items = (res.items || []) as HomelabPkgApisNetworkSiteV1SyncPolicy[];
      if (reset) {
        this.policies.set(items);
      } else {
        const current = this.policies();
        const newItems = items.filter((n) => !current.some((e) => e.id === n.id));
        this.policies.update((prev) => [...prev, ...newItems]);
      }
      this.total.set(items.length);
      this.nextCursor.set(res.nextCursor || '');
      this.hasMore.set(res.hasMore || false);
    } catch (err) {
      console.error(err);
    } finally {
      this.loading.set(false);
      this.loadingMore.set(false);
    }
  }

  createPolicy() {
    requestAnimationFrame(() => {
      const dialogRef = this.dialog.open(CreateSyncPolicyDialogComponent, {
        width: '500px',
        data: {},
      });
      dialogRef.afterClosed().subscribe((res) => {
        if (res) this.loadPolicies(true);
      });
    });
  }

  editPolicy(policy: HomelabPkgApisNetworkSiteV1SyncPolicy) {
    requestAnimationFrame(() => {
      const dialogRef = this.dialog.open(CreateSyncPolicyDialogComponent, {
        width: '500px',
        data: { policy },
      });
      dialogRef.afterClosed().subscribe((res) => {
        if (res) this.loadPolicies(true);
      });
    });
  }

  async togglePolicy(policy: HomelabPkgApisNetworkSiteV1SyncPolicy) {
    if (!policy.id) return;
    this.loading.set(true);
    try {
      const updated = {
        ...policy,
        meta: { ...policy.meta, enabled: !policy.meta?.enabled },
      };
      await firstValueFrom(this.siteService.networkSiteSyncIdPut(policy.id, updated));
      this.snackBar.open(updated.meta.enabled ? '策略已启用' : '策略已禁用', '关闭', {
        duration: 2000,
      });
      await this.loadPolicies(true);
    } catch (err: any) {
      this.snackBar.open(`操作失败: ${err.error?.message || err.message}`, '关闭', {
        duration: 3000,
      });
    } finally {
      this.loading.set(false);
    }
  }

  deletePolicy(policy: HomelabPkgApisNetworkSiteV1SyncPolicy) {
    requestAnimationFrame(() => {
      const dialogRef = this.dialog.open(ConfirmDialogComponent, {
        data: {
          title: '删除确认',
          message: `确定要删除策略 [${policy.meta?.name}] 吗？`,
          confirmText: '确定删除',
          color: 'warn',
        },
      });
      dialogRef.afterClosed().subscribe(async (res) => {
        if (res && policy.id) {
          this.loading.set(true);
          try {
            await firstValueFrom(this.siteService.networkSiteSyncIdDelete(policy.id));
            this.snackBar.open('删除成功', '关闭', { duration: 3000 });
            this.loadPolicies(true);
          } catch (err: any) {
            this.snackBar.open(`删除失败: ${err.error?.message || err.message}`, '关闭', {
              duration: 3000,
            });
          } finally {
            this.loading.set(false);
          }
        }
      });
    });
  }

  async triggerSync(policy: HomelabPkgApisNetworkSiteV1SyncPolicy) {
    if (!policy.id) return;
    this.syncingRows.update((s) => ({ ...s, [policy.id!]: true }));
    try {
      await firstValueFrom(this.siteService.networkSiteSyncIdTriggerPost(policy.id));
      this.snackBar.open('同步任务已触发', '关闭', { duration: 2000 });
      await this.loadPolicies(true);
    } catch (err: any) {
      this.snackBar.open(`同步失败: ${err.error?.message || err.message}`, '关闭', {
        duration: 3000,
      });
    } finally {
      this.syncingRows.update((s) => ({ ...s, [policy.id!]: false }));
    }
  }
}
