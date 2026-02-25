import { Component, Inject, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MAT_DIALOG_DATA, MatDialogModule } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { FormsModule } from '@angular/forms';
import { MatIconModule } from '@angular/material/icon';
import { AuthRole, AuthPolicyRule } from '../../generated';

@Component({
  selector: 'app-create-role-dialog',
  standalone: true,
  imports: [
    CommonModule,
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    FormsModule,
    MatIconModule,
  ],
  template: `
    <h2 mat-dialog-title class="!pt-6">{{ isEdit ? '修改 Role' : '创建 Role' }}</h2>
    <mat-dialog-content>
      <div class="pt-3 space-y-5">
        <mat-form-field appearance="outline" class="w-full">
          <mat-label>Role 名称</mat-label>
          <input
            matInput
            [(ngModel)]="role.name"
            placeholder="例如: dns-admin"
            [disabled]="isEdit"
            autofocus
          />
          <mat-error *ngIf="!isEdit && isDuplicate()">角色名称已存在</mat-error>
        </mat-form-field>

        <div
          class="border border-outline-variant rounded-xl p-4 space-y-4 bg-surface-container-low"
          *ngIf="firstRule"
        >
          <p class="text-xs font-bold text-outline uppercase tracking-wider">权限规则</p>

          <mat-form-field appearance="outline" class="w-full bg-surface">
            <mat-label>资源 (Resources)</mat-label>
            <input
              matInput
              [(ngModel)]="resourceInput"
              placeholder="dns, service 或 * (回车添加)"
              (keyup.enter)="addResource()"
            />
            <mat-hint>输入资源名称并按回车</mat-hint>
          </mat-form-field>
          <div class="flex flex-wrap gap-2">
            <span
              *ngFor="let r of firstRule.resources"
              class="bg-primary-container text-on-primary-container px-3 py-1 rounded-full text-xs font-medium flex items-center gap-1.5 shadow-sm"
            >
              {{ r }}
              <mat-icon
                class="!w-[14px] !h-[14px] !text-[14px] !flex !items-center !justify-center cursor-pointer opacity-70 hover:opacity-100"
                (click)="removeResource(r)"
                >close</mat-icon
              >
            </span>
          </div>

          <mat-form-field appearance="outline" class="w-full bg-surface">
            <mat-label>操作 (Verbs)</mat-label>
            <input
              matInput
              [(ngModel)]="verbInput"
              placeholder="read, write 或 * (回车添加)"
              (keyup.enter)="addVerb()"
            />
            <mat-hint>输入动词并按回车</mat-hint>
          </mat-form-field>
          <div class="flex flex-wrap gap-2">
            <span
              *ngFor="let v of firstRule.verbs"
              class="bg-secondary-container text-on-secondary-container px-3 py-1 rounded-full text-xs font-medium flex items-center gap-1.5 shadow-sm"
            >
              {{ v }}
              <mat-icon
                class="!w-[14px] !h-[14px] !text-[14px] !flex !items-center !justify-center cursor-pointer opacity-70 hover:opacity-100"
                (click)="removeVerb(v)"
                >close</mat-icon
              >
            </span>
          </div>
        </div>
      </div>
    </mat-dialog-content>
    <mat-dialog-actions align="end" class="!px-6 !pb-6">
      <button mat-button mat-dialog-close>取消</button>
      <button
        mat-flat-button
        color="primary"
        [mat-dialog-close]="role"
        [disabled]="
          !role.name ||
          !firstRule ||
          (firstRule.resources?.length || 0) === 0 ||
          (firstRule.verbs?.length || 0) === 0 ||
          (!isEdit && isDuplicate())
        "
        class="!ml-2"
      >
        {{ isEdit ? '保存修改' : '确认创建' }}
      </button>
    </mat-dialog-actions>
  `,
})
export class CreateRoleDialogComponent {
  isEdit = false;
  role: AuthRole = {
    name: '',
    rules: [{ verbs: [], resources: [] }],
  };
  existingNames: string[] = [];

  resourceInput = '';
  verbInput = '';

  constructor(
    @Inject(MAT_DIALOG_DATA) public data: { role: AuthRole | null; existingNames?: string[] },
  ) {
    if (data.role) {
      this.isEdit = true;
      this.role = JSON.parse(JSON.stringify(data.role));
      if (!this.role.rules || this.role.rules.length === 0) {
        this.role.rules = [{ verbs: [], resources: [] }];
      }
    }
    this.existingNames = data.existingNames || [];
  }

  isDuplicate(): boolean {
    return this.existingNames.includes(this.role.name?.trim() || '');
  }

  get firstRule(): AuthPolicyRule | undefined {
    return this.role.rules && this.role.rules.length > 0 ? this.role.rules[0] : undefined;
  }

  addResource() {
    const val = this.resourceInput.trim();
    const rule = this.firstRule;
    if (val && rule) {
      if (!rule.resources) rule.resources = [];
      if (!rule.resources.includes(val)) {
        rule.resources.push(val);
        this.resourceInput = '';
      }
    }
  }

  removeResource(r: string) {
    const rule = this.firstRule;
    if (rule && rule.resources) {
      rule.resources = rule.resources.filter((x) => x !== r);
    }
  }

  addVerb() {
    const val = this.verbInput.trim();
    const rule = this.firstRule;
    if (val && rule) {
      if (!rule.verbs) rule.verbs = [];
      if (!rule.verbs.includes(val)) {
        rule.verbs.push(val);
        this.verbInput = '';
      }
    }
  }

  removeVerb(v: string) {
    const rule = this.firstRule;
    if (rule && rule.verbs) {
      rule.verbs = rule.verbs.filter((x) => x !== v);
    }
  }
}
