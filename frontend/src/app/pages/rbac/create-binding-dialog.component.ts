import { Component, Inject, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MAT_DIALOG_DATA, MatDialogModule } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import {
  MatAutocompleteModule,
  MatAutocompleteSelectedEvent,
} from '@angular/material/autocomplete';
import { MatChipsModule } from '@angular/material/chips';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { MatIconModule } from '@angular/material/icon';
import { MatDividerModule } from '@angular/material/divider';
import { FormsModule } from '@angular/forms';
import { ModelsServiceAccount, ModelsRole, ModelsRoleBinding, RbacService } from '../../generated';
import {
  firstValueFrom,
  Subject,
  debounceTime,
  distinctUntilChanged,
  switchMap,
  of,
  catchError,
} from 'rxjs';

@Component({
  selector: 'app-create-binding-dialog',
  standalone: true,
  imports: [
    CommonModule,
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    MatAutocompleteModule,
    MatChipsModule,
    MatSlideToggleModule,
    MatIconModule,
    MatDividerModule,
    FormsModule,
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

        <!-- ServiceAccount Autocomplete -->
        <mat-form-field appearance="outline" class="w-full">
          <mat-label>目标服务账号 (ServiceAccount)</mat-label>
          <input
            matInput
            [matAutocomplete]="saAuto"
            [(ngModel)]="saSearchValue"
            (input)="onSaSearch($any($event.target).value)"
            placeholder="搜索账号 ID 或名称..."
            [disabled]="isEdit"
            required
          />
          <mat-autocomplete
            #saAuto="matAutocomplete"
            [displayWith]="displaySaFn.bind(this)"
            (optionSelected)="onSaSelected($event)"
          >
            @for (sa of filteredSa(); track sa.id) {
              <mat-option [value]="sa">
                <div class="flex flex-col py-1">
                  <span class="font-medium">{{ sa.name || '未命名账号' }}</span>
                  <span class="text-[10px] text-outline font-mono">{{ sa.id }}</span>
                </div>
              </mat-option>
            }
          </mat-autocomplete>
          <mat-hint
            >当前已选 ID:
            <code class="font-bold text-primary">{{
              binding.serviceAccountId || '未选择'
            }}</code></mat-hint
          >
        </mat-form-field>

        <!-- Roles Chip Grid with Autocomplete -->
        <mat-form-field appearance="outline" class="w-full">
          <mat-label>赋予角色 (Roles)</mat-label>
          <mat-chip-grid #chipGrid>
            @for (roleId of binding.roleIds; track roleId) {
              <mat-chip-row (removed)="removeRole(roleId)" class="!bg-secondary-container">
                <div class="flex flex-col leading-tight py-0.5">
                  <span class="text-[10px] font-bold">{{ getRoleName(roleId) }}</span>
                  <span class="text-[8px] opacity-60 font-mono">{{ roleId.slice(0, 8) }}...</span>
                </div>
                <button matChipRemove><mat-icon>cancel</mat-icon></button>
              </mat-chip-row>
            }
            <input
              placeholder="搜索并添加角色..."
              [matAutocomplete]="roleAuto"
              [matChipInputFor]="chipGrid"
              (input)="onRoleSearch($any($event.target).value)"
              #roleInput
            />
          </mat-chip-grid>
          <mat-autocomplete
            #roleAuto="matAutocomplete"
            (optionSelected)="onRoleSelected($event, roleInput)"
          >
            @for (role of filteredRoles(); track role.id) {
              <mat-option [value]="role">
                <div class="flex flex-col py-1">
                  <span class="font-medium">{{ role.name }}</span>
                  <span class="text-[10px] text-outline font-mono">{{ role.id }}</span>
                </div>
              </mat-option>
            }
          </mat-autocomplete>
          <mat-hint>可搜索并添加多个角色</mat-hint>
        </mat-form-field>

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
  private rbacService = inject(RbacService);
  isEdit = false;
  binding: ModelsRoleBinding = {
    id: '',
    name: '',
    serviceAccountId: '',
    roleIds: [],
    enabled: true,
  };

  // Search and Filtering
  saSearchValue = '';
  filteredSa = signal<ModelsServiceAccount[]>([]);
  filteredRoles = signal<ModelsRole[]>([]);

  // Store full objects for name mapping in chips
  private roleLookup = new Map<string, string>();

  constructor(
    @Inject(MAT_DIALOG_DATA)
    public data: {
      binding?: ModelsRoleBinding;
      serviceAccounts: ModelsServiceAccount[];
      roles: ModelsRole[];
    },
  ) {
    if (data.binding) {
      this.isEdit = true;
      this.binding = JSON.parse(JSON.stringify(data.binding));
      if (!this.binding.roleIds) this.binding.roleIds = [];
    }

    // Initialize lookup from current cache
    data.roles.forEach((r) => this.roleLookup.set(r.id!, r.name!));
    this.filteredSa.set(data.serviceAccounts.slice(0, 50));
    this.filteredRoles.set(data.roles.slice(0, 50));

    if (this.isEdit && this.binding.serviceAccountId) {
      const sa = data.serviceAccounts.find((s) => s.id === this.binding.serviceAccountId);
      if (sa) this.saSearchValue = sa.name || sa.id || '';
    }
  }

  ngOnInit() {}

  displaySaFn(sa: any): string {
    if (typeof sa === 'string') return sa;
    return sa ? sa.name || sa.id : '';
  }

  async onSaSearch(val: string) {
    if (typeof val !== 'string') return;
    const res = await firstValueFrom(this.rbacService.rbacServiceaccountsGet(1, 50, val));
    this.filteredSa.set(res.items || []);
  }

  onSaSelected(event: MatAutocompleteSelectedEvent) {
    const sa = event.option.value as ModelsServiceAccount;
    this.binding.serviceAccountId = sa.id || '';
    this.saSearchValue = sa.name || sa.id || '';
  }

  async onRoleSearch(val: string) {
    if (typeof val !== 'string') return;
    const res = await firstValueFrom(this.rbacService.rbacRolesGet(1, 50, val));
    this.filteredRoles.set(res.items || []);
    // Update lookup for new roles found
    res.items?.forEach((r) => this.roleLookup.set(r.id!, r.name!));
  }

  onRoleSelected(event: MatAutocompleteSelectedEvent, input: HTMLInputElement) {
    const role = event.option.value as ModelsRole;
    if (role.id && !this.binding.roleIds?.includes(role.id)) {
      this.binding.roleIds = [...(this.binding.roleIds || []), role.id];
      this.roleLookup.set(role.id, role.name!);
    }
    input.value = '';
    this.onRoleSearch(''); // Reset suggestions
  }

  removeRole(id: string) {
    this.binding.roleIds = this.binding.roleIds?.filter((rid) => rid !== id);
  }

  getRoleName(id: string): string {
    return this.roleLookup.get(id) || id;
  }
}
