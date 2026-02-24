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
    <h2 mat-dialog-title>ServiceAccount 已创建</h2>
    <mat-dialog-content>
      <div class="space-y-4">
        <p>这是您的 ServiceAccount <strong>{{ data.name }}</strong> 的 Token。</p>
        <div class="bg-amber-50 border border-amber-200 rounded-lg p-4 space-y-3">
          <div class="flex items-start gap-2 text-amber-800">
            <mat-icon class="mt-1 scale-75">warning</mat-icon>
            <p class="text-sm font-medium">请务必保存此 Token，它将不再显示！</p>
          </div>
          <div class="flex items-center gap-2 bg-white border rounded p-2 font-mono text-sm break-all">
            <span class="flex-1">{{ data.token }}</span>
            <button mat-icon-button (click)="copyToken()">
              <mat-icon>content_copy</mat-icon>
            </button>
          </div>
        </div>
      </div>
    </mat-dialog-content>
    <mat-dialog-actions align="end">
      <button mat-flat-button color="primary" mat-dialog-close>我已保存</button>
    </mat-dialog-actions>
  `,
})
export class ShowTokenDialogComponent {
  private snackBar = inject(MatSnackBar);
  constructor(@Inject(MAT_DIALOG_DATA) public data: { name: string, token: string }) {}

  copyToken() {
    navigator.clipboard.writeText(this.data.token);
    this.snackBar.open('Token 已复制到剪贴板', '关闭', { duration: 2000 });
  }
}
