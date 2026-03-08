import { Component, Inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MAT_DIALOG_DATA, MatDialogModule } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { ModelsAuditLog } from '../../generated';

@Component({
  selector: 'app-audit-detail-dialog',
  standalone: true,
  imports: [CommonModule, MatDialogModule, MatButtonModule, MatIconModule],
  template: `
    <h2 mat-dialog-title class="pt-6!">
      <mat-icon class="mr-2 align-middle text-primary">history</mat-icon>
      审计日志详情
    </h2>
    <mat-dialog-content style="min-width: 350px; max-width: 700px;">
      <div class="pt-2 space-y-6">
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div class="space-y-1">
            <span class="text-[10px] font-bold text-outline uppercase tracking-widest"
              >操作时间</span
            >
            <div class="text-sm font-medium">
              {{ data.timestamp | date: 'yyyy-MM-dd HH:mm:ss' }}
            </div>
          </div>
          <div class="space-y-1">
            <span class="text-[10px] font-bold text-outline uppercase tracking-widest"
              >操作状态</span
            >
            <div>
              <span
                [class]="
                  'px-2 py-0.5 rounded-full text-[10px] font-bold ' +
                  (data.status === 'Success'
                    ? 'bg-primary-container text-on-primary-container'
                    : 'bg-error/10 text-error')
                "
              >
                {{ data.status }}
              </span>
            </div>
          </div>
          <div class="space-y-1">
            <span class="text-[10px] font-bold text-outline uppercase tracking-widest"
              >操作人 (Subject)</span
            >
            <div class="text-sm font-mono text-primary">{{ data.subject }}</div>
          </div>
          <div class="space-y-1">
            <span class="text-[10px] font-bold text-outline uppercase tracking-widest"
              >操作动作 (Action)</span
            >
            <div class="text-sm font-bold">{{ data.action }}</div>
          </div>
          <div class="space-y-1">
            <span class="text-[10px] font-bold text-outline uppercase tracking-widest"
              >资源模块</span
            >
            <div class="text-sm">{{ data.resource }}</div>
          </div>
          <div class="space-y-1">
            <span class="text-[10px] font-bold text-outline uppercase tracking-widest"
              >目标 ID</span
            >
            <div class="text-sm font-mono">{{ data.targetId || '-' }}</div>
          </div>
        </div>

        <div class="space-y-1">
          <span class="text-[10px] font-bold text-outline uppercase tracking-widest"
            >操作描述 (Message)</span
          >
          <div
            class="text-sm p-4 bg-surface-container rounded-2xl border border-outline-variant/30 leading-relaxed"
          >
            {{ data.message || '无详细描述' }}
          </div>
        </div>

        <div class="grid grid-cols-1 gap-4">
          <div class="space-y-1">
            <span class="text-[10px] font-bold text-outline uppercase tracking-widest"
              >IP 地址</span
            >
            <div class="text-sm font-mono">{{ data.ipAddress }}</div>
          </div>
          <div class="space-y-1">
            <span class="text-[10px] font-bold text-outline uppercase tracking-widest"
              >User Agent</span
            >
            <div class="text-[10px] font-mono opacity-60 break-all leading-normal">
              {{ data.userAgent }}
            </div>
          </div>
        </div>
      </div>
    </mat-dialog-content>
    <mat-dialog-actions align="end" class="px-6! pb-6!">
      <button mat-flat-button color="primary" mat-dialog-close class="px-8 rounded-full">
        关闭
      </button>
    </mat-dialog-actions>
  `,
})
export class AuditDetailDialogComponent {
  constructor(@Inject(MAT_DIALOG_DATA) public data: ModelsAuditLog) {}
}
