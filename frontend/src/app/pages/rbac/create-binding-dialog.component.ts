import { Component, Inject, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MAT_DIALOG_DATA, MatDialogModule } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { MatIconModule } from '@angular/material/icon';
import { MatDividerModule } from '@angular/material/divider';
import { FormsModule } from '@angular/forms';
import { ModelsServiceAccount, ModelsRole, ModelsRoleBinding, RbacService } from '../../generated';
import { firstValueFrom } from 'rxjs';

@Component({
  selector: 'app-create-binding-dialog',
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
    MatDividerModule,
    FormsModule,
  ],
  template: `
    <h2 mat-dialog-title class="!pt-6">{{ isEdit ? '修改 RoleBinding' : '创建 RoleBinding' }}</h2>
    <mat-dialog-content style="min-width: 350px; max-width: 500px;">
      <div class="pt-3 space-y-5">
        <mat-form-field appearance="outline" class="w-full">
          <mat-label>绑定名称</mat-label>
          <input
            matInput
            [(ngModel)]="binding.name"
            placeholder="例如: backup-agent-dns-admin"
            [disabled]="isEdit"
            autofocus
          />
          <mat-error *ngIf="!isEdit && isDuplicate()">绑定名称已存在</mat-error>
        </mat-form-field>

        <!-- ServiceAccount Select with Pagination -->
        <mat-form-field appearance="outline" class="w-full">
          <mat-label>ServiceAccount</mat-label>
          <mat-select [(ngModel)]="binding.serviceAccountName" [disabled]="isEdit">
            <mat-option *ngFor="let sa of serviceAccounts()" [value]="sa.name">
              {{ sa.name }}
            </mat-option>
            <div *ngIf="hasMoreSa()" class="px-2 py-1">
              <button
                mat-button
                class="w-full !text-xs !text-primary"
                (click)="$event.stopPropagation(); loadMoreSa()"
              >
                加载更多...
              </button>
            </div>
          </mat-select>
        </mat-form-field>

        <!-- Roles Select with Pagination -->
        <mat-form-field appearance="outline" class="w-full">
          <mat-label>赋予角色 (Roles)</mat-label>
          <mat-select [(ngModel)]="binding.roleNames" multiple>
            <mat-option *ngFor="let role of roles()" [value]="role.name">
              {{ role.name }}
            </mat-option>
            <div *ngIf="hasMoreRoles()" class="px-2 py-1">
              <button
                mat-button
                class="w-full !text-xs !text-primary"
                (click)="$event.stopPropagation(); loadMoreRoles()"
              >
                加载更多...
              </button>
            </div>
          </mat-select>
          <mat-hint>可选择一个或多个角色</mat-hint>
        </mat-form-field>

        <div
          class="flex items-center justify-between p-4 bg-surface rounded-xl border border-outline-variant"
        >
          <div class="flex flex-col">
            <span class="text-sm font-medium">启用此绑定</span>
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
          !binding.serviceAccountName ||
          !binding.roleNames ||
          binding.roleNames.length === 0 ||
          (!isEdit && isDuplicate())
        "
        class="!ml-2"
      >
        {{ isEdit ? '保存修改' : '确认创建' }}
      </button>
    </mat-dialog-actions>
  `,
})
export class CreateBindingDialogComponent implements OnInit {
  private rbacService = inject(RbacService);
  isEdit = false;
  binding: ModelsRoleBinding = {
    name: '',
    serviceAccountName: '',
    roleNames: [],
    enabled: true,
  };
  existingNames: string[] = [];

  serviceAccounts = signal<ModelsServiceAccount[]>([]);
  roles = signal<ModelsRole[]>([]);

  saPage = 0;
  rolePage = 0;
  pageSize = 20;

  hasMoreSa = signal(false);
  hasMoreRoles = signal(false);

  constructor(
    @Inject(MAT_DIALOG_DATA) public data: { binding?: ModelsRoleBinding; existingNames?: string[] },
  ) {
    if (data.binding) {
      this.isEdit = true;
      this.binding = { ...data.binding };
      if (!this.binding.roleNames) {
        this.binding.roleNames = [];
      }
    } else {
      this.binding.enabled = true;
    }
    this.existingNames = data.existingNames || [];
  }

  ngOnInit() {
    this.loadInitialData();
  }

  async loadInitialData() {
    await Promise.all([this.fetchSa(true), this.fetchRoles(true)]);
  }

  async fetchSa(reset = false) {
    if (reset) {
      this.saPage = 0;
      this.serviceAccounts.set([]);
    }
    // Note: Backend ListPage prefix is not used yet for contains search, but we fetch all and filter client side
    // or we fetch first page and support pagination.
    // Since backend uses prefix, we can't do full 'contains' search easily with prefix.
    // I'll fetch a larger page size for dropdowns.
    const res = await firstValueFrom(
      this.rbacService.rbacServiceaccountsGet(this.saPage + 1, this.pageSize),
    );
    const items = res.items || [];

    if (reset) {
      this.serviceAccounts.set(items);
    } else {
      this.serviceAccounts.update((prev) => [...prev, ...items]);
    }

    this.hasMoreSa.set(this.serviceAccounts().length < (res.total || 0));
  }

  async fetchRoles(reset = false) {
    if (reset) {
      this.rolePage = 0;
      this.roles.set([]);
    }
    const res = await firstValueFrom(
      this.rbacService.rbacRolesGet(this.rolePage + 1, this.pageSize),
    );
    const items = res.items || [];

    if (reset) {
      this.roles.set(items);
    } else {
      this.roles.update((prev) => [...prev, ...items]);
    }

    this.hasMoreRoles.set(this.roles().length < (res.total || 0));
  }

  loadMoreSa() {
    this.saPage++;
    this.fetchSa();
  }

  loadMoreRoles() {
    this.rolePage++;
    this.fetchRoles();
  }

  isDuplicate(): boolean {
    return this.existingNames.includes(this.binding.name?.trim() || '');
  }
}
