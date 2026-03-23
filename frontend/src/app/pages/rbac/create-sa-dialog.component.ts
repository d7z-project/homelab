import { Component, Inject, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { MatIconModule } from '@angular/material/icon';
import { FormsModule } from '@angular/forms';
import { ModelsServiceAccount } from '../../generated';

@Component({
  selector: 'app-create-sa-dialog',
  standalone: true,
  imports: [
    CommonModule,
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    MatSlideToggleModule,
    MatIconModule,
    FormsModule,
  ],
  template: `
    <h2 mat-dialog-title class="pt-6!">
      <mat-icon class="mr-2 align-middle text-primary">person_outline</mat-icon>
      {{ isEdit ? '修改 ServiceAccount' : '创建 ServiceAccount' }}
    </h2>
    <mat-dialog-content style="min-width: 320px; max-width: 500px;">
      <div class="pt-3 space-y-5">
        <mat-form-field appearance="outline" class="w-full">
          <mat-label>账号 ID (唯一标识)</mat-label>
          <input
            matInput
            [(ngModel)]="sa.id"
            placeholder="例如: backup-agent"
            [disabled]="isEdit"
            autofocus
            required
            pattern="^[a-zA-Z0-9_\\-]+$"
            #idInput="ngModel"
          />
          @if (!isEdit) {
            <mat-hint>仅允许字母、数字、中划线和下划线</mat-hint>
          }
          @if (!isEdit && idInput.errors?.['pattern']) {
            <mat-error>ID 格式不正确</mat-error>
          }
          @if (!isEdit && isDuplicate()) {
            <mat-error>ID 已存在</mat-error>
          }
        </mat-form-field>

        <mat-form-field appearance="outline" class="w-full">
          <mat-label>显示名称</mat-label>
          <input
            matInput
            [(ngModel)]="sa.meta.name"
            placeholder="例如: 备份代理"
            (keyup.enter)="confirm()"
          />
        </mat-form-field>

        <mat-form-field appearance="outline" class="w-full">
          <mat-label>备注 (Comments)</mat-label>
          <textarea
            matInput
            [(ngModel)]="sa.meta.comments"
            placeholder="说明此账号的用途..."
            rows="3"
          ></textarea>
        </mat-form-field>

        <div
          class="flex items-center justify-between p-4 bg-surface-container-low rounded-2xl border border-outline-variant/30"
        >
          <div class="flex flex-col">
            <span class="text-sm font-bold">启用此账号</span>
            <span class="text-xs text-outline">禁用后使用该 ID 的所有 API 访问将被拒绝</span>
          </div>
          <mat-slide-toggle color="primary" [(ngModel)]="sa.meta.enabled"></mat-slide-toggle>
        </div>
      </div>
    </mat-dialog-content>
    <mat-dialog-actions align="end" class="px-6! pb-6!">
      <button mat-button mat-dialog-close>取消</button>
      <button
        mat-flat-button
        color="primary"
        (click)="confirm()"
        [disabled]="!sa.id.trim() || (!isEdit && (isDuplicate() || idInput.errors?.['pattern']))"
        class="ml-2! px-8 rounded-full"
      >
        <mat-icon class="mr-1">check</mat-icon>
        {{ isEdit ? '保存修改' : '确认创建' }}
      </button>
    </mat-dialog-actions>
  `,
})
export class CreateSaDialogComponent {
  private dialogRef = inject(MatDialogRef<CreateSaDialogComponent>);
  isEdit = false;
  sa: ModelsServiceAccount = {
    id: '',
    meta: {
      name: '',
      comments: '',
      enabled: true,
    },
    status: {} as any,
  };
  existingIDs: string[] = [];

  constructor(
    @Inject(MAT_DIALOG_DATA)
    public data: { sa: ModelsServiceAccount | null; existingIDs?: string[] },
  ) {
    if (data.sa) {
      this.isEdit = true;
      this.sa = {
        ...data.sa,
        meta: { ...data.sa.meta },
      };
    }
    this.existingIDs = data.existingIDs || [];
  }

  isDuplicate(): boolean {
    return this.existingIDs.includes(this.sa.id.trim());
  }

  confirm() {
    if (this.sa.id.trim() && (this.isEdit || !this.isDuplicate())) {
      this.dialogRef.close(this.sa);
    }
  }
}
