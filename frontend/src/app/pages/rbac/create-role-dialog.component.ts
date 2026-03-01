import { Component, Inject, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MAT_DIALOG_DATA, MatDialogModule } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatIconModule } from '@angular/material/icon';
import { MatAutocompleteModule, MatAutocompleteSelectedEvent } from '@angular/material/autocomplete';
import { FormsModule } from '@angular/forms';
import { AuthRole, AuthPolicyRule, RbacService } from '../../generated';
import { firstValueFrom } from 'rxjs';

@Component({
  selector: 'app-create-role-dialog',
  standalone: true,
  imports: [
    CommonModule,
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    MatIconModule,
    MatAutocompleteModule,
    FormsModule,
  ],
  template: `
    <h2 mat-dialog-title>{{ isEdit ? '编辑角色' : '创建角色' }}</h2>
    <mat-dialog-content class="flex flex-col gap-4 min-w-[400px]">
      <mat-form-field appearance="outline" class="w-full">
        <mat-label>角色名称</mat-label>
        <input matInput [(ngModel)]="role.name" [disabled]="isEdit" placeholder="例如: admin, viewer" />
        <mat-error *ngIf="isDuplicate()">角色名称已存在</mat-error>
      </mat-form-field>

      <div *ngIf="firstRule" class="space-y-4 border border-outline-variant/30 p-4 rounded-2xl bg-surface-container-lowest">
        <div class="text-xs font-bold text-outline uppercase tracking-wider mb-2">权限规则</div>

        <div class="flex flex-col gap-4">
          <mat-form-field appearance="outline" class="w-full bg-surface">
            <mat-label>资源 (Resources)</mat-label>
            <input
              matInput
              [(ngModel)]="resourceInput"
              [matAutocomplete]="auto"
              placeholder="dns, dns/**, dns/example.com 或 * (回车添加)"
              (input)="onResourceInput($event)"
              (keyup.enter)="addResource()"
            />
            <mat-autocomplete #auto="matAutocomplete" (optionSelected)="onSuggestionSelected($event)">
              <mat-option *ngFor="let suggestion of suggestions()" [value]="suggestion">
                {{ suggestion }}
              </mat-option>
            </mat-autocomplete>
            <mat-hint>输入并从下拉列表选择，或按回车添加。支持 dns/example.com 格式</mat-hint>
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
            <mat-hint>常用动作: read, write, create, delete, *</mat-hint>
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
    <mat-dialog-actions align="end" class="p-4">
      <button mat-button mat-dialog-close>取消</button>
      <button
        mat-flat-button
        color="primary"
        [mat-dialog-close]="role"
        [disabled]="!role.name || isDuplicate() || !firstRule?.resources?.length || !firstRule?.verbs?.length"
      >
        确定
      </button>
    </mat-dialog-actions>
  `,
})
export class CreateRoleDialogComponent {
  private rbacService = inject(RbacService);
  
  isEdit = false;
  role: AuthRole = {
    name: '',
    rules: [{ verbs: [], resources: [] }],
  };
  existingNames: string[] = [];

  resourceInput = '';
  verbInput = '';
  
  suggestions = signal<string[]>([]);

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
    if (this.isEdit) return false;
    return this.existingNames.includes(this.role.name?.trim() || '');
  }

  get firstRule(): AuthPolicyRule | undefined {
    return this.role.rules && this.role.rules.length > 0 ? this.role.rules[0] : undefined;
  }

  async onResourceInput(event: any) {
    const val = this.resourceInput.trim();
    try {
      const list = await firstValueFrom(this.rbacService.rbacResourcesSuggestGet(val));
      this.suggestions.set(list || []);
    } catch (e) {
      this.suggestions.set([]);
    }
  }

  onSuggestionSelected(event: MatAutocompleteSelectedEvent) {
    this.resourceInput = event.option.viewValue;
    this.addResource();
  }

  addResource() {
    const val = this.resourceInput.trim();
    const rule = this.firstRule;
    if (val && rule) {
      if (!rule.resources) rule.resources = [];
      if (!rule.resources.includes(val)) {
        rule.resources.push(val);
        this.resourceInput = '';
        this.suggestions.set([]);
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
