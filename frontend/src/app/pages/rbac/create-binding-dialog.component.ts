import { Component, Inject, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MAT_DIALOG_DATA, MatDialogModule } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { FormsModule } from '@angular/forms';
import { AuthServiceAccount, AuthRole, AuthRoleBinding } from '../../generated';

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
    FormsModule,
  ],
  template: `
    <h2 mat-dialog-title>{{ isEdit ? '修改 RoleBinding' : '创建 RoleBinding' }}</h2>
    <mat-dialog-content>
      <div class="pt-2 space-y-4">
        <mat-form-field appearance="outline" class="w-full">
          <mat-label>绑定名称</mat-label>
          <input matInput [(ngModel)]="binding.name" placeholder="例如: backup-agent-dns-admin" [disabled]="isEdit" autofocus />
          <mat-error *ngIf="!isEdit && isDuplicate()">绑定名称已存在</mat-error>
        </mat-form-field>

        <mat-form-field appearance="outline" class="w-full">
          <mat-label>ServiceAccount</mat-label>
          <mat-select [(ngModel)]="binding.serviceAccountName">
            <mat-option *ngFor="let sa of data.serviceAccounts" [value]="sa.name">
              {{sa.name}}
            </mat-option>
          </mat-select>
        </mat-form-field>

        <mat-form-field appearance="outline" class="w-full">
          <mat-label>Role</mat-label>
          <mat-select [(ngModel)]="binding.roleName">
            <mat-option *ngFor="let role of data.roles" [value]="role.name">
              {{role.name}}
            </mat-option>
          </mat-select>
        </mat-form-field>
      </div>
    </mat-dialog-content>
    <mat-dialog-actions align="end">
      <button mat-button mat-dialog-close>取消</button>
      <button mat-flat-button color="primary" [mat-dialog-close]="binding" 
              [disabled]="!binding.name || !binding.serviceAccountName || !binding.roleName || (!isEdit && isDuplicate())">
        {{ isEdit ? '保存修改' : '确认创建' }}
      </button>
    </mat-dialog-actions>
  `,
})
export class CreateBindingDialogComponent {
  isEdit = false;
  binding = {
    name: '',
    serviceAccountName: '',
    roleName: ''
  };
  existingNames: string[] = [];

  constructor(@Inject(MAT_DIALOG_DATA) public data: { serviceAccounts: AuthServiceAccount[], roles: AuthRole[], binding?: AuthRoleBinding, existingNames?: string[] }) {
    if (data.binding) {
      this.isEdit = true;
      this.binding = { ...data.binding } as any;
    }
    this.existingNames = data.existingNames || [];
  }

  isDuplicate(): boolean {
    return this.existingNames.includes(this.binding.name?.trim() || '');
  }
}
