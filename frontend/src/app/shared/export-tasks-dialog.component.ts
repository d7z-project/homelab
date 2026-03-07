import { Component, Inject, OnInit, OnDestroy, signal, computed, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';
import { FormsModule } from '@angular/forms';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { NetworkIpService, NetworkSiteService } from '../generated';
import { firstValueFrom, interval, Subscription } from 'rxjs';

@Component({
  selector: 'app-export-tasks-dialog',
  standalone: true,
  imports: [
    CommonModule,
    MatDialogModule,
    MatButtonModule,
    MatIconModule,
    MatProgressBarModule,
    MatTooltipModule,
    MatInputModule,
    MatFormFieldModule,
    FormsModule,
    MatSnackBarModule,
  ],
  template: `
    <h2 mat-dialog-title class="!flex items-center justify-between">
      <div class="flex items-center gap-2">
        <mat-icon color="primary">history</mat-icon>
        <span>导出任务列表 ({{ type === 'ip' ? 'IP' : '域名' }})</span>
      </div>
      <button mat-icon-button mat-dialog-close><mat-icon>close</mat-icon></button>
    </h2>

    <mat-dialog-content class="!p-0 min-h-[400px]">
      <div class="px-6 py-4 border-b border-outline-variant/30 sticky top-0 z-10 bg-surface">
        <mat-form-field appearance="outline" class="w-full search-field-m3" subscriptSizing="dynamic">
          <mat-label>搜索任务 ID 或状态...</mat-label>
          <input matInput [(ngModel)]="search" placeholder="输入关键字过滤..." />
          <mat-icon matPrefix class="mr-2 opacity-60">search</mat-icon>
          @if (search()) {
            <button mat-icon-button matSuffix (click)="search.set('')"><mat-icon>close</mat-icon></button>
          }
        </mat-form-field>
      </div>

      <div class="flex flex-col divide-y divide-outline-variant/20 bg-surface">
        @for (task of filteredTasks(); track task.ID) {
          <div class="px-6 py-4 hover:bg-surface-container-low transition-colors group">
            <div class="flex items-start justify-between gap-4 mb-2">
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2 mb-1">
                  <span class="font-mono text-sm font-bold truncate max-w-[200px]" [matTooltip]="task.ID">
                    {{ task.ID }}
                  </span>
                  <span
                    class="px-2 py-0.5 rounded-md text-[10px] font-bold border uppercase tracking-wider"
                    [class.bg-blue-50]="task.Status === 'Running' || task.Status === 'Pending'"
                    [class.text-blue-700]="task.Status === 'Running' || task.Status === 'Pending'"
                    [class.border-blue-200]="task.Status === 'Running' || task.Status === 'Pending'"
                    [class.bg-green-50]="task.Status === 'Success'"
                    [class.text-green-700]="task.Status === 'Success'"
                    [class.border-green-200]="task.Status === 'Success'"
                    [class.bg-red-50]="task.Status === 'Failed' || task.Status === 'Cancelled'"
                    [class.text-red-700]="task.Status === 'Failed' || task.Status === 'Cancelled'"
                    [class.border-red-200]="task.Status === 'Failed' || task.Status === 'Cancelled'"
                  >
                    {{ task.Status }}
                  </span>
                </div>
                <div class="flex items-center gap-4 text-xs text-outline opacity-80 mt-2">
                  <div class="flex items-center gap-1">
                    <mat-icon class="!w-4 !h-4 !text-[16px]">calendar_today</mat-icon>
                    <span>{{ task.CreatedAt | date: 'yyyy-MM-dd HH:mm:ss' }}</span>
                  </div>
                  <div class="flex items-center gap-1 font-bold uppercase">
                    <mat-icon class="!w-4 !h-4 !text-[16px]">description</mat-icon>
                    <span>{{ task.Format | uppercase }}</span>
                  </div>                  <div class="flex items-center gap-1 font-mono font-medium">
                    <mat-icon class="!w-4 !h-4 !text-[16px]">data_array</mat-icon>
                    <span>{{ task.RecordCount || 0 }} 条数据</span>
                  </div>
                </div>
              </div>

              <div class="flex items-center gap-2">
                @if (task.Status === 'Success') {
                  <button
                    mat-icon-button
                    color="primary"
                    (click)="copyLink(task.ResultURL)"
                    matTooltip="复制下载地址"
                  >
                    <mat-icon>content_copy</mat-icon>
                  </button>
                  <a
                    mat-flat-button
                    color="primary"
                    [href]="task.ResultURL"
                    target="_blank"
                    class="!rounded-xl"
                  >
                    <mat-icon class="mr-1">download</mat-icon>
                    下载
                  </a>
                }
                @if (task.Status === 'Failed') {
                  <button mat-icon-button color="warn" [matTooltip]="task.Error">
                    <mat-icon>error_outline</mat-icon>
                  </button>
                }
              </div>
            </div>

            @if (task.Status === 'Running' || task.Status === 'Pending') {
              <div class="space-y-1 mt-3">
                <mat-progress-bar
                  mode="determinate"
                  [value]="(task.Progress || 0) * 100"
                  class="h-2 rounded-full"
                ></mat-progress-bar>
                <div class="flex justify-end text-xs font-mono text-primary font-bold">
                  {{ (task.Progress || 0) * 100 | number: '1.0-1' }}%
                </div>
              </div>
            }
          </div>
        } @empty {
          <div class="p-16 flex flex-col items-center justify-center text-outline/40 italic">
            <mat-icon class="!w-16 !h-16 !text-[64px] mb-4 opacity-20">cloud_off</mat-icon>
            <span class="text-base">暂无匹配的导出任务</span>
          </div>
        }
      </div>
    </mat-dialog-content>

    <div mat-dialog-actions align="end" class="!px-6 !py-4 border-t border-outline-variant/30">
      <div class="flex-1 text-xs text-outline text-left italic">
        任务将在 24 小时后自动清理
      </div>
      <button mat-button mat-dialog-close class="px-6 !rounded-xl">关闭</button>
    </div>
  `,  styles: [
    `
      :host {
        display: block;
      }
      mat-dialog-content::-webkit-scrollbar {
        width: 6px;
      }
      mat-dialog-content::-webkit-scrollbar-thumb {
        background: rgba(0, 0, 0, 0.1);
        border-radius: 10px;
      }
      .overflow-y-auto::-webkit-scrollbar {
        width: 6px;
      }
      .overflow-y-auto::-webkit-scrollbar-thumb {
        background: rgba(0, 0, 0, 0.1);
        border-radius: 10px;
      }
      .search-field-m3 {
        ::ng-deep .mdc-text-field--filled {
          background-color: transparent !important;
        }
        ::ng-deep .mdc-line-ripple {
          display: none;
        }
        ::ng-deep .mat-mdc-text-field-wrapper {
          padding-bottom: 0;
        }
      }
    `,
  ],
})
export class ExportTasksDialogComponent implements OnInit, OnDestroy {
  private ipService = inject(NetworkIpService);
  private siteService = inject(NetworkSiteService);
  private snackBar = inject(MatSnackBar);

  type: 'ip' | 'site' = 'ip';
  tasks = signal<any[]>([]);
  search = signal('');

  private pollSub?: Subscription;

  filteredTasks = computed(() => {
    const s = this.search().toLowerCase();
    return this.tasks()
      .filter((t) => t.ID.toLowerCase().includes(s) || t.Status.toLowerCase().includes(s))
      .sort((a, b) => new Date(b.CreatedAt).getTime() - new Date(a.CreatedAt).getTime());
  });

  constructor(
    public dialogRef: MatDialogRef<ExportTasksDialogComponent>,
    @Inject(MAT_DIALOG_DATA) public data: { type: 'ip' | 'site' }
  ) {
    this.type = data.type;
  }

  ngOnInit() {
    this.refresh();
    this.pollSub = interval(2000).subscribe(() => this.refresh());
  }

  ngOnDestroy() {
    this.pollSub?.unsubscribe();
  }

  async refresh() {
    try {
      let res: any[];
      if (this.type === 'ip') {
        res = await firstValueFrom(this.ipService.networkIpExportsTasksGet());
      } else {
        res = await firstValueFrom(this.siteService.networkSiteExportsTasksGet());
      }
      this.tasks.set(res || []);
    } catch (err) {
      console.error('Failed to load export tasks:', err);
    }
  }

  copyLink(url: string) {
    const fullUrl = window.location.origin + url;
    navigator.clipboard.writeText(fullUrl).then(() => {
      this.snackBar.open('已复制下载地址', '关闭', { duration: 2000 });
    });
  }
}
