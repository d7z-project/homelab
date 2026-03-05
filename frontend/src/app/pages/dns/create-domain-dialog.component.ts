import { Component, Inject, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { MatIconModule } from '@angular/material/icon';
import { FormsModule } from '@angular/forms';
import { ModelsDomain } from '../../generated';

@Component({
  selector: 'app-create-domain-dialog',
  standalone: true,
  imports: [
    CommonModule,
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatSlideToggleModule,
    MatIconModule,
    FormsModule,
  ],
  template: `
    <h2 mat-dialog-title class="!pt-6">
      <mat-icon class="mr-2 align-middle text-primary">language</mat-icon>
      {{ isEdit ? '修改域名配置' : '添加新域名' }}
    </h2>
    <mat-dialog-content style="min-width: 320px; max-width: 500px;">
      <div class="pt-3 space-y-6">
        <mat-form-field appearance="outline" class="w-full">
          <mat-label>域名名称 (FQDN)</mat-label>
          <input
            matInput
            [(ngModel)]="domain.name"
            #nameInput="ngModel"
            placeholder="例如: example.com"
            [disabled]="isEdit"
            autofocus
            (keyup.enter)="confirm()"
            required
            pattern="^([\\-a-zA-Z0-9]+([\\-a-zA-Z0-9]+)*\\.)+[a-zA-Z]{2,}$"
          />
          @if (!isEdit) {
            <mat-hint>创建后名称不可直接修改</mat-hint>
          }
          @if (!isEdit && nameInput.errors?.['required']) {
            <mat-error>请输入域名名称</mat-error>
          }
          @if (!isEdit && nameInput.errors?.['pattern']) {
            <mat-error>域名格式不正确</mat-error>
          }
          @if (!isEdit && isDuplicate()) {
            <mat-error>域名已存在</mat-error>
          }
        </mat-form-field>

        <div
          class="flex items-center justify-between p-4 bg-surface-container-low rounded-2xl border border-outline-variant/30"
        >
          <div class="flex flex-col">
            <span class="text-sm font-bold">解析状态</span>
            <span class="text-xs text-outline">禁用后该域名下的所有记录将停止解析</span>
          </div>
          <mat-slide-toggle color="primary" [(ngModel)]="domain.enabled"> </mat-slide-toggle>
        </div>

        <mat-form-field appearance="outline" class="w-full">
          <mat-label>备注信息</mat-label>
          <textarea
            matInput
            [(ngModel)]="domain.comments"
            placeholder="说明此域名的用途..."
            rows="3"
          ></textarea>
        </mat-form-field>
      </div>
    </mat-dialog-content>
    <mat-dialog-actions align="end" class="!px-6 !pb-6">
      <button mat-button mat-dialog-close>取消</button>
      <button
        mat-flat-button
        color="primary"
        (click)="confirm()"
        [disabled]="!isValid()"
        class="!ml-2 px-6 rounded-full"
      >
        <mat-icon class="mr-1">check</mat-icon>
        {{ isEdit ? '保存更改' : '立即创建' }}
      </button>
    </mat-dialog-actions>
  `,
})
export class CreateDomainDialogComponent {
  private dialogRef = inject(MatDialogRef<CreateDomainDialogComponent>);
  isEdit = false;
  domain: ModelsDomain = {
    name: '',
    enabled: true,
    comments: '',
  };
  existingNames: string[] = [];

  constructor(
    @Inject(MAT_DIALOG_DATA)
    public data: { domain: ModelsDomain | null; existingNames?: string[] },
  ) {
    if (data.domain) {
      this.isEdit = true;
      this.domain = { ...data.domain };
    }
    this.existingNames = data.existingNames || [];
  }

  isDuplicate(): boolean {
    const name = this.domain.name?.trim().toLowerCase();
    return this.existingNames.some((n) => n.toLowerCase() === name);
  }

  isValid(): boolean {
    const name = this.domain.name?.trim();
    if (!name) return false;
    if (!this.isEdit && this.isDuplicate()) return false;
    // Simple regex for domain validation
    return /^([\-a-zA-Z0-9]+([\-a-zA-Z0-9]+)*\.)+[a-zA-Z]{2,}$/.test(name.toLowerCase());
  }

  confirm() {
    if (this.isValid()) {
      this.domain.name = this.domain.name?.toLowerCase().trim();
      this.dialogRef.close(this.domain);
    }
  }
}
