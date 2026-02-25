import { Component, Inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MAT_DIALOG_DATA, MatDialogModule } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';

export interface ConfirmDialogData {
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  color?: 'primary' | 'accent' | 'warn';
}

@Component({
  selector: 'app-confirm-dialog',
  standalone: true,
  imports: [CommonModule, MatDialogModule, MatButtonModule, MatIconModule],
  template: `
    <h2 mat-dialog-title class="flex items-center gap-3 !pt-6">
      <mat-icon [color]="data.color || 'warn'" class="!w-6 !h-6 !text-[24px]">warning</mat-icon>
      {{ data.title }}
    </h2>
    <mat-dialog-content>
      <p class="py-3 text-on-surface opacity-80">{{ data.message }}</p>
    </mat-dialog-content>
    <mat-dialog-actions align="end" class="!px-6 !pb-6">
      <button mat-button [mat-dialog-close]="false">{{ data.cancelText || '取消' }}</button>
      <button mat-flat-button [color]="data.color || 'warn'" [mat-dialog-close]="true" class="!ml-2">
        {{ data.confirmText || '确定删除' }}
      </button>
    </mat-dialog-actions>
  `,
})
export class ConfirmDialogComponent {
  constructor(@Inject(MAT_DIALOG_DATA) public data: ConfirmDialogData) {}
}
