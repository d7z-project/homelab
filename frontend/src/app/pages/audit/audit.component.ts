import { Component, OnInit, inject, signal, computed, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatCardModule } from '@angular/material/card';
import { MatTableModule } from '@angular/material/table';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { MatChipsModule } from '@angular/material/chips';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatDialog, MatDialogModule } from '@angular/material/dialog';
import { AuditService, ModelsAuditLog, AuthService, ControllersAuthInfo } from '../../generated';
import { firstValueFrom } from 'rxjs';
import { UiService } from '../../ui.service';
import { MatSnackBar } from '@angular/material/snack-bar';
import { AuditDetailDialogComponent } from './audit-detail-dialog.component';
import { BreakpointObserver, Breakpoints } from '@angular/cdk/layout';
import { toSignal } from '@angular/core/rxjs-interop';
import { map } from 'rxjs/operators';
import { ConfirmDialogComponent } from '../rbac/confirm-dialog.component';

@Component({
  selector: 'app-audit',
  standalone: true,
  imports: [
    CommonModule,
    MatCardModule,
    MatTableModule,
    MatIconModule,
    MatButtonModule,
    MatChipsModule,
    MatTooltipModule,
    MatProgressSpinnerModule,
    MatDialogModule,
  ],
  templateUrl: './audit.component.html',
})
export class AuditComponent implements OnInit, OnDestroy {
  private auditService = inject(AuditService);
  private authService = inject(AuthService);
  private snackBar = inject(MatSnackBar);
  private dialog = inject(MatDialog);
  public uiService = inject(UiService);
  private breakpointObserver = inject(BreakpointObserver);

  private scrollListener?: () => void;

  isHandset = toSignal(
    this.breakpointObserver.observe(Breakpoints.Handset).pipe(map((result) => result.matches)),
    { initialValue: this.breakpointObserver.isMatched(Breakpoints.Handset) },
  );

  logs = signal<ModelsAuditLog[]>([]);
  total = signal(0);
  page = signal(1);
  pageSize = signal(20);
  search = signal('');
  loading = signal(false);
  loadingMore = signal(false);
  showScrollTop = signal(false);

  // User auth state
  authInfo = signal<ControllersAuthInfo | null>(null);
  isRoot = computed(() => this.authInfo()?.type === 'root');

  displayedColumns = computed(() =>
    this.isHandset()
      ? ['timestamp', 'subject', 'action', 'status']
      : ['timestamp', 'subject', 'action', 'resource', 'targetId', 'status', 'actions'],
  );

  fabConfig = computed(() => {
    if (this.isRoot()) {
      return {
        icon: 'delete_sweep',
        label: '清理日志',
        action: () => this.cleanup(),
        color: 'warn' as const,
      };
    }
    return null;
  });

  constructor() {}

  ngOnInit(): void {
    this.loadAuthInfo();
    this.loadLogs(true);
    this.setupScrollListener();
  }

  ngOnDestroy(): void {
    if (this.scrollListener) {
      const scrollElement = document.querySelector('mat-sidenav-content');
      scrollElement?.removeEventListener('scroll', this.scrollListener);
    }
  }

  async loadAuthInfo() {
    try {
      const info = await firstValueFrom(this.authService.infoGet());
      this.authInfo.set(info);
    } catch (e) {}
  }

  private setupScrollListener() {
    const scrollElement = document.querySelector('mat-sidenav-content');
    if (!scrollElement) return;

    this.scrollListener = () => {
      this.showScrollTop.set(scrollElement.scrollTop > 300);
      const atBottom =
        scrollElement.scrollHeight - scrollElement.scrollTop <= scrollElement.clientHeight + 150;

      if (atBottom && this.hasMore() && !this.loadingMore() && !this.loading()) {
        this.loadMore();
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

  async loadLogs(reset = false) {
    if (reset) {
      this.loading.set(true);
      this.page.set(1);
    }

    try {
      const res = await firstValueFrom(
        this.auditService.auditLogsGet(this.page(), this.pageSize(), this.search()),
      );
      if (reset) {
        this.logs.set(res.items || []);
      } else {
        const currentLogs = this.logs();
        const newItems = (res.items || []).filter(
          (newItem) => !currentLogs.some((existing) => existing.id === newItem.id),
        );
        this.logs.update((prev) => [...prev, ...newItems]);
      }
      this.total.set(res.total || 0);
    } catch (err) {
      this.snackBar
        .open('加载审计日志失败', '重试', { duration: 3000 })
        .onAction()
        .subscribe(() => this.loadLogs(reset));
    } finally {
      this.loading.set(false);
      this.loadingMore.set(false);
    }
  }

  loadMore() {
    if (this.loadingMore() || !this.hasMore()) return;
    this.loadingMore.set(true);
    this.page.update((p) => p + 1);
    this.loadLogs();
  }

  hasMore = computed(() => this.logs().length < this.total());

  onSearch(term: string) {
    this.search.set(term);
    this.loadLogs(true);
  }

  openSearch() {
    this.uiService.openSearch({
      placeholder: '搜索操作人、动作、资源或目标...',
      value: this.search(),
      onSearch: (v) => this.onSearch(v),
    });
  }

  showDetail(log: ModelsAuditLog, event?: Event) {
    if (event) {
      event.stopPropagation();
    }
    this.dialog.open(AuditDetailDialogComponent, {
      data: log,
      maxWidth: '90vw',
      width: '600px',
    });
  }

  async cleanup() {
    requestAnimationFrame(() => {
      const dialogRef = this.dialog.open(ConfirmDialogComponent, {
        data: {
          title: '清理审计日志',
          message: '确定要清理 30 天前的所有历史日志吗？此操作不可撤销。',
          confirmText: '清理 30 天前日志',
          color: 'warn',
        },
      });

      dialogRef.afterClosed().subscribe(async (result) => {
        if (result) {
          this.loading.set(true);
          try {
            const res = await firstValueFrom(this.auditService.auditLogsCleanupPost(30));
            this.snackBar.open(`清理成功，已删除 ${res.deleted} 条记录`, '关闭', {
              duration: 3000,
            });
            this.loadLogs(true);
          } catch (err: any) {
            this.snackBar.open(err.error?.message || '清理失败', '关闭', { duration: 3000 });
          } finally {
            this.loading.set(false);
          }
        }
      });
    });
  }
}
