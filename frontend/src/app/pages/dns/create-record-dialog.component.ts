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
import { ModelsDomain, ModelsRecord } from '../../generated';

@Component({
  selector: 'app-create-record-dialog',
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
      <mat-icon class="mr-2 align-middle text-primary">layers</mat-icon>
      {{ isEdit ? '修改解析记录' : '新增解析记录' }}
    </h2>
    <mat-dialog-content style="min-width: 350px; max-width: 600px;">
      <div class="pt-3 space-y-4">
        <mat-form-field appearance="outline" class="w-full">
          <mat-label>所属域名</mat-label>
          <mat-select [(ngModel)]="record.domainId" [disabled]="isEdit">
            <mat-option *ngFor="let d of domains" [value]="d.id">{{ d.name }}</mat-option>
          </mat-select>
        </mat-form-field>

        <div class="flex gap-4">
          <mat-form-field appearance="outline" class="flex-1">
            <mat-label>主机记录 (Name)</mat-label>
            <input matInput [(ngModel)]="record.name" placeholder="例如: www 或 @" />
            <mat-hint>@ 表示主域名</mat-hint>
          </mat-form-field>

          <mat-form-field appearance="outline" class="w-32">
            <mat-label>记录类型</mat-label>
            <mat-select [(ngModel)]="record.type">
              <mat-option *ngFor="let t of recordTypes" [value]="t">{{ t }}</mat-option>
            </mat-select>
          </mat-form-field>
        </div>

        <mat-form-field appearance="outline" class="w-full">
          <mat-label>记录值 (Value)</mat-label>
          <input matInput [(ngModel)]="record.value" [placeholder]="getValuePlaceholder()" />
        </mat-form-field>

        <div class="flex gap-4">
          <mat-form-field appearance="outline" class="flex-1">
            <mat-label>TTL (秒)</mat-label>
            <input matInput type="number" [(ngModel)]="record.ttl" min="1" placeholder="默认 600" />
            <mat-hint>推荐值: 600, 3600, 86400</mat-hint>
          </mat-form-field>

          <mat-form-field
            *ngIf="record.type === 'MX' || record.type === 'SRV'"
            appearance="outline"
            class="w-32"
          >
            <mat-label>优先级</mat-label>
            <input matInput type="number" [(ngModel)]="record.priority" />
          </mat-form-field>
        </div>

        <div class="flex items-center justify-between px-4 py-3 bg-surface-container rounded-2xl">
          <div class="flex flex-col">
            <span class="font-medium">启用状态</span>
            <span class="text-xs text-outline">禁用后此条记录将不再参与解析</span>
          </div>
          <mat-slide-toggle
            color="primary"
            [checked]="record.status === 'active'"
            (change)="record.status = $event.checked ? 'active' : 'inactive'"
          >
          </mat-slide-toggle>
        </div>

        <div
          *ngIf="record.type === 'CNAME'"
          class="p-3 bg-warn-container text-on-warn-container rounded-xl text-xs flex gap-2"
        >
          <mat-icon class="text-sm h-4 w-4">info</mat-icon>
          <span>提示: CNAME 记录不能与同一主机记录下的其他记录（如 A, TXT）共存。</span>
        </div>
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
        {{ isEdit ? '保存更改' : '确定添加' }}
      </button>
    </mat-dialog-actions>
  `,
})
export class CreateRecordDialogComponent {
  private dialogRef = inject(MatDialogRef<CreateRecordDialogComponent>);
  isEdit = false;
  record: ModelsRecord = {
    domainId: '',
    name: '',
    type: 'A',
    value: '',
    ttl: 600,
    priority: 10,
    status: 'active',
  };
  domains: ModelsDomain[] = [];
  recordTypes = ['A', 'AAAA', 'CNAME', 'MX', 'TXT', 'NS', 'SRV', 'CAA'];

  constructor(
    @Inject(MAT_DIALOG_DATA)
    public data: { record: ModelsRecord | null; domains: ModelsDomain[]; defaultDomainId?: string },
  ) {
    this.domains = data.domains || [];
    if (data.record) {
      this.isEdit = true;
      this.record = { ...data.record };
    } else if (data.defaultDomainId) {
      this.record.domainId = data.defaultDomainId;
    } else if (this.domains.length > 0) {
      this.record.domainId = this.domains[0].id || '';
    }
  }

  getValuePlaceholder(): string {
    switch (this.record.type) {
      case 'A':
        return 'IPv4 地址, 如 1.2.3.4';
      case 'AAAA':
        return 'IPv6 地址, 如 2001:db8::1';
      case 'CNAME':
        return '别名域名, 如 example.com.';
      case 'MX':
        return '邮件服务器, 如 mail.example.com.';
      default:
        return '记录内容...';
    }
  }

  isValid(): boolean {
    return !!(this.record.domainId && this.record.name && this.record.type && this.record.value);
  }

  confirm() {
    if (this.isValid()) {
      this.dialogRef.close(this.record);
    }
  }
}
