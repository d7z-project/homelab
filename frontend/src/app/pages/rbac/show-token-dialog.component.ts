import { Component, Inject, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MAT_DIALOG_DATA, MatDialogModule } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatSnackBar } from '@angular/material/snack-bar';

@Component({
  selector: 'app-show-token-dialog',
  standalone: true,
  imports: [CommonModule, MatDialogModule, MatButtonModule, MatIconModule],
  template: `
    <h2 mat-dialog-title class="pt-6!">ServiceAccount 已就绪</h2>
    <mat-dialog-content style="min-width: 350px; max-width: 550px;">
      <div class="space-y-4 pt-2">
        <div class="flex flex-col gap-1 text-on-surface opacity-80">
          <p>
            账号 ID: <strong class="font-mono text-primary">{{ data.id }}</strong>
          </p>
          <p>
            显示名称: <strong>{{ data.name || '-' }}</strong>
          </p>
        </div>
        <div
          class="bg-error-container/10 rounded-2xl p-5 space-y-4 border border-error/30 animate-in shake-x duration-500"
        >
          <div class="flex items-start gap-3 text-error">
            <mat-icon
              class="w-[20px]! h-[20px]! text-[20px]! flex! items-center! justify-center! shrink-0"
              >warning</mat-icon
            >
            <div class="space-y-1">
              <p class="text-xs font-bold leading-relaxed uppercase tracking-wider">
                一次性机密令牌
              </p>
              <p class="text-[11px] opacity-80 leading-relaxed">
                后端仅存储此令牌的哈希摘要。出于安全考虑，离开此页面后将<strong>无法再次找回</strong>。泄露或遗失请立即重置。
              </p>
            </div>
          </div>
          <div
            class="flex items-center gap-3 bg-surface-container-lowest border border-error/20 rounded-xl p-3 font-mono text-xs break-all shadow-inner"
          >
            <span class="flex-1 select-all text-on-surface">{{ data.token }}</span>
            <button
              mat-icon-button
              (click)="copyToken()"
              class="text-error flex items-center justify-center hover:bg-error/10 transition-colors"
            >
              <mat-icon class="w-[20px]! h-[20px]! text-[20px]! flex! items-center! justify-center!"
                >content_copy</mat-icon
              >
            </button>
          </div>
        </div>
      </div>
    </mat-dialog-content>
    <mat-dialog-actions align="end" class="px-6! pb-6!">
      <button mat-flat-button color="primary" mat-dialog-close>已安全保存</button>
    </mat-dialog-actions>
  `,
})
export class ShowTokenDialogComponent {
  private snackBar = inject(MatSnackBar);
  constructor(@Inject(MAT_DIALOG_DATA) public data: { id: string; name: string; token: string }) {}

  copyToken() {
    navigator.clipboard.writeText(this.data.token);
    this.snackBar.open('Token 已复制到剪贴板', '关闭', { duration: 2000 });
  }
}
