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
    <h2 mat-dialog-title class="!pt-6">ServiceAccount 已就绪</h2>
    <mat-dialog-content>
      <div class="space-y-4 pt-2">
        <p class="text-on-surface opacity-80">
          账号 <strong>{{ data.name }}</strong> 的访问令牌已生成。
        </p>
        <div class="bg-surface-container rounded-2xl p-5 space-y-4 border border-outline-variant">
          <div class="flex items-start gap-3 text-primary">
            <mat-icon
              class="!w-[20px] !h-[20px] !text-[20px] !flex !items-center !justify-center shrink-0"
              >info</mat-icon
            >
            <p class="text-xs font-medium leading-relaxed">
              请妥善保管此令牌。出于安全考虑，离开此页面后将无法再次查看该令牌。
            </p>
          </div>
          <div
            class="flex items-center gap-3 bg-surface-container-lowest border border-outline-variant rounded-xl p-3 font-mono text-xs break-all shadow-inner"
          >
            <span class="flex-1 select-all">{{ data.token }}</span>
            <button
              mat-icon-button
              (click)="copyToken()"
              class="text-primary flex items-center justify-center"
            >
              <mat-icon class="!w-[20px] !h-[20px] !text-[20px] !flex !items-center !justify-center"
                >content_copy</mat-icon
              >
            </button>
          </div>
        </div>
      </div>
    </mat-dialog-content>
    <mat-dialog-actions align="end" class="!px-6 !pb-6">
      <button mat-flat-button color="primary" mat-dialog-close>已安全保存</button>
    </mat-dialog-actions>
  `,
})
export class ShowTokenDialogComponent {
  private snackBar = inject(MatSnackBar);
  constructor(@Inject(MAT_DIALOG_DATA) public data: { name: string; token: string }) {}

  copyToken() {
    navigator.clipboard.writeText(this.data.token);
    this.snackBar.open('Token 已复制到剪贴板', '关闭', { duration: 2000 });
  }
}
