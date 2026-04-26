import { Component, OnInit, inject, signal, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatTableModule } from '@angular/material/table';
import { MatDialog, MatDialogModule } from '@angular/material/dialog';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { NetworkIntelligenceService, V1Source } from '../../../generated';
import { PageHeaderComponent } from '../../../shared/page-header.component';
import { CreateSourceDialogComponent } from './create-source-dialog.component';
import { ConfirmDialogComponent } from '../../rbac/confirm-dialog.component';

@Component({
  selector: 'app-intelligence',
  standalone: true,
  imports: [
    CommonModule,
    MatCardModule,
    MatButtonModule,
    MatIconModule,
    MatTableModule,
    MatDialogModule,
    MatTooltipModule,
    MatProgressSpinnerModule,
    MatProgressBarModule,
    PageHeaderComponent,
  ],
  templateUrl: './intelligence.component.html',
})
export class IntelligenceComponent implements OnInit, OnDestroy {
  private intService = inject(NetworkIntelligenceService);
  private dialog = inject(MatDialog);
  private snackBar = inject(MatSnackBar);

  sources = signal<V1Source[]>([]);
  loading = signal(false);
  private pollInterval: any;

  displayedColumns = ['type', 'name', 'url', 'cron', 'status', 'lastUpdated', 'actions'];

  ngOnInit() {
    this.loadSources();
    // Poll status every 3 seconds if any is downloading
    this.pollInterval = setInterval(() => {
      const data = this.sources();
      if (
        Array.isArray(data) &&
        data.some((s) => s.status?.status === 'Running' || s.status?.status === 'Pending')
      ) {
        this.loadSources();
      }
    }, 3000);
  }

  ngOnDestroy() {
    if (this.pollInterval) {
      clearInterval(this.pollInterval);
    }
  }

  loadSources() {
    this.loading.set(true);
    this.intService.networkIntelligenceSourcesGet('', 100).subscribe({
      next: (res) => {
        this.sources.set((res.items || []) as V1Source[]);
        this.loading.set(false);
      },
      error: (err) => {
        console.error('Failed to load intelligence sources', err);
        this.sources.set([]);
        this.loading.set(false);
      },
    });
  }

  createSource() {
    this.dialog
      .open(CreateSourceDialogComponent, { width: '500px' })
      .afterClosed()
      .subscribe((res) => {
        if (res) this.loadSources();
      });
  }

  editSource(source: V1Source) {
    this.dialog
      .open(CreateSourceDialogComponent, { width: '500px', data: source })
      .afterClosed()
      .subscribe((res) => {
        if (res) this.loadSources();
      });
  }

  syncSource(source: V1Source) {
    if (!source.id) return;
    this.intService.networkIntelligenceSourcesIdSyncPost(source.id).subscribe({
      next: () => {
        this.snackBar.open('同步任务已启动', '关闭', { duration: 2000 });
        this.loadSources();
      },
      error: (err) =>
        this.snackBar.open(`触发失败: ${err.error?.message || err.message}`, '关闭', {
          duration: 3000,
        }),
    });
  }

  deleteSource(source: V1Source) {
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      data: { title: '删除确认', message: `确定要删除情报源 [${source.meta?.name}] 吗？` },
    });
    dialogRef.afterClosed().subscribe((res) => {
      if (res && source.id) {
        this.intService.networkIntelligenceSourcesIdDelete(source.id).subscribe({
          next: () => {
            this.snackBar.open('删除成功', '关闭', { duration: 3000 });
            this.loadSources();
          },
          error: (err) =>
            this.snackBar.open(`删除失败: ${err.error?.message || err.message}`, '关闭', {
              duration: 3000,
            }),
        });
      }
    });
  }
}
