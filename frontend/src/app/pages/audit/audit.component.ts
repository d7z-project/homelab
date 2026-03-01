import { Component, OnInit, inject, signal, computed, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatTableModule } from '@angular/material/table';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatButtonModule } from '@angular/material/button';
import { MatTooltipModule } from '@angular/material/tooltip';
import { FormsModule } from '@angular/forms';
import { AuditService, ModelsAuditLog } from '../../generated';
import { firstValueFrom } from 'rxjs';
import { MatSnackBar } from '@angular/material/snack-bar';
import { UiService } from '../../ui.service';
import { MatDialog, MatDialogModule } from '@angular/material/dialog';
import { BreakpointObserver, Breakpoints } from '@angular/cdk/layout';
import { toSignal } from '@angular/core/rxjs-interop';
import { map } from 'rxjs/operators';
import { AuditDetailDialogComponent } from './audit-detail-dialog.component';

@Component({
  selector: 'app-audit',
  standalone: true,
  imports: [
    CommonModule,
    MatTableModule,
    MatIconModule,
    MatProgressSpinnerModule,
    MatButtonModule,
    MatTooltipModule,
    MatDialogModule,
    FormsModule,
  ],
  templateUrl: './audit.component.html',
})
export class AuditComponent implements OnInit {
  private auditService = inject(AuditService);
  private snackBar = inject(MatSnackBar);
  public uiService = inject(UiService);
  private dialog = inject(MatDialog);
  private breakpointObserver = inject(BreakpointObserver);

  isHandset = toSignal(
    this.breakpointObserver.observe(Breakpoints.Handset).pipe(map((result) => result.matches)),
    { initialValue: this.breakpointObserver.isMatched(Breakpoints.Handset) },
  );

  logs = signal<ModelsAuditLog[]>([]);
  total = signal(0);
  page = signal(0);
  pageSize = signal(50);
  loading = signal(false);
  loadingMore = signal(false);

  search = signal('');

  // Use computed for table columns to ensure stability and proper initialization
  displayedColumns = computed(() =>
    this.isHandset()
      ? ['timestamp', 'action', 'resource', 'status']
      : ['timestamp', 'subject', 'action', 'resource', 'targetId', 'message', 'status'],
  );

  constructor() {}

  hasMore = computed(() => this.logs().length < this.total());

  openSearch() {
    this.uiService.openSearch({
      placeholder: '搜索审计日志...',
      value: this.search(),
      onSearch: (v) => {
        this.search.set(v);
        this.refresh();
      },
    });
  }

  ngOnInit() {
    this.refresh();
  }

  onSearchChange(event: any) {
    this.refresh();
  }

  clearSearch() {
    this.search.set('');
    this.refresh();
  }

  async refresh() {
    this.loading.set(true);
    this.page.set(0);
    try {
      const data = await firstValueFrom(
        this.auditService.auditLogsGet(this.page(), this.pageSize(), this.search()),
      );
      this.logs.set(data.items || []);
      this.total.set(data.total || 0);
    } catch (err) {
      this.snackBar.open('加载日志失败', '关闭', { duration: 3000 });
    } finally {
      this.loading.set(false);
    }
  }

  async loadMore() {
    if (!this.hasMore()) return;
    this.loadingMore.set(true);
    this.page.update((p) => p + 1);
    try {
      const data = await firstValueFrom(
        this.auditService.auditLogsGet(this.page(), this.pageSize(), this.search()),
      );
      this.logs.update((prev) => [...prev, ...(data.items || [])]);
    } catch (err) {
      this.page.update((p) => p - 1);
      this.snackBar.open('加载更多失败', '关闭', { duration: 3000 });
    } finally {
      this.loadingMore.set(false);
    }
  }

  showDetail(log: ModelsAuditLog) {
    requestAnimationFrame(() => {
      this.dialog.open(AuditDetailDialogComponent, {
        maxWidth: '95vw',
        data: log,
      });
    });
  }
}
