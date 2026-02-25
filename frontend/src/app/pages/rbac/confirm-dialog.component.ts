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
    <h2 mat-dialog-title class="!pt-6">{{ data.title }}</h2>
    <mat-dialog-content>
      <p class="py-3 text-on-surface opacity-80 leading-relaxed">{{ data.message }}</p>
    </mat-dialog-content>
    <mat-dialog-actions align="end" class="!px-6 !pb-6 !gap-2">
      <button mat-button [mat-dialog-close]="false" class="!rounded-full px-4">
        {{ data.cancelText || '取消' }}
      </button>
      <button
        mat-flat-button
        [color]="data.color === 'warn' || !data.color ? null : data.color"
        [class.bg-error]="data.color === 'warn' || !data.color"
        [class.text-on-error]="data.color === 'warn' || !data.color"
        [mat-dialog-close]="true"
        class="!rounded-full px-6"
      >
        <mat-icon
          *ngIf="data.color !== 'primary'"
          class="mr-1.5 !w-5 !h-5 !text-[20px] !text-inherit"
          >delete_outline</mat-icon
        >
        {{ data.confirmText || '确定删除' }}
      </button>
    </mat-dialog-actions>
  `,
})
export class ConfirmDialogComponent {
  constructor(@Inject(MAT_DIALOG_DATA) public data: ConfirmDialogData) {}
}
