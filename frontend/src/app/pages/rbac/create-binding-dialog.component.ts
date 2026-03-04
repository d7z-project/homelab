import { Component, Inject, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MAT_DIALOG_DATA, MatDialogModule } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { MatIconModule } from '@angular/material/icon';
import { MatDividerModule } from '@angular/material/divider';
import { FormsModule } from '@angular/forms';
import { ModelsServiceAccount, ModelsRole, ModelsRoleBinding, RbacService } from '../../generated';
import { DiscoverySelectComponent } from '../../shared/discovery-select.component';

@Component({
  selector: 'app-create-binding-dialog',
  standalone: true,
  imports: [
    CommonModule,
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    MatSlideToggleModule,
    MatIconModule,
    MatDividerModule,
    FormsModule,
    DiscoverySelectComponent,
  ],
  template: `
    <h2 mat-dialog-title class="!pt-6">{{ isEdit ? '修改权限绑定' : '创建权限绑定' }}</h2>
    <mat-dialog-content style="min-width: 400px; max-width: 600px;">
      <div class="pt-3 space-y-6">
        @if (isEdit) {
          <mat-form-field appearance="outline" class="w-full" subscriptSizing="dynamic">
            <mat-label>绑定 ID (只读)</mat-label>
            <input matInput [value]="binding.id" disabled />
          </mat-form-field>
        }

        <mat-form-field appearance="outline" class="w-full">
          <mat-label>显示名称</mat-label>
          <input
            matInput
            [(ngModel)]="binding.name"
            placeholder="例如: 备份代理权限绑定"
            required
          />
        </mat-form-field>

        <!-- ServiceAccount Discovery Select -->
        <app-discovery-select
          code="rbac/serviceaccounts"
          label="目标服务账号 (ServiceAccount)"
          placeholder="搜索账号 ID 或名称..."
          [(ngModel)]="binding.serviceAccountId"
          [disabled]="isEdit"
        ></app-discovery-select>

        <!-- Roles Discovery Select (Multiple) -->
        <app-discovery-select
          code="rbac/roles"
          label="赋予角色 (Roles)"
          placeholder="搜索并添加角色..."
          [(ngModel)]="binding.roleIds"
          [multiple]="true"
          hint="可搜索并添加多个角色"
        ></app-discovery-select>

        <div
          class="flex items-center justify-between p-4 bg-surface-container-low rounded-2xl border border-outline-variant/30"
        >
          <div class="flex flex-col">
            <span class="text-sm font-bold">启用此绑定</span>
            <span class="text-xs text-outline">禁用后此账号将暂时失去该组权限</span>
          </div>
          <mat-slide-toggle color="primary" [(ngModel)]="binding.enabled"></mat-slide-toggle>
        </div>
      </div>
    </mat-dialog-content>
    <mat-dialog-actions align="end" class="!px-6 !pb-6">
      <button mat-button mat-dialog-close>取消</button>
      <button
        mat-flat-button
        color="primary"
        [mat-dialog-close]="binding"
        [disabled]="
          !binding.name ||
          !binding.serviceAccountId ||
          !binding.roleIds ||
          binding.roleIds.length === 0
        "
        class="!ml-2 px-8 rounded-full"
      >
        <mat-icon class="mr-1">check</mat-icon>
        {{ isEdit ? '保存修改' : '确认创建' }}
      </button>
    </mat-dialog-actions>
  `,
})
export class CreateBindingDialogComponent implements OnInit {
  isEdit = false;
  binding: ModelsRoleBinding = {
    id: '',
    name: '',
    serviceAccountId: '',
    roleIds: [],
    enabled: true,
  };

  constructor(
    @Inject(MAT_DIALOG_DATA)
    public data: {
      binding?: ModelsRoleBinding;
    },
  ) {
    if (data.binding) {
      this.isEdit = true;
      this.binding = JSON.parse(JSON.stringify(data.binding));
      if (!this.binding.roleIds) this.binding.roleIds = [];
    }
  }

  ngOnInit() {}
}
