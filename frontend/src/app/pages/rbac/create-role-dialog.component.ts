import { Component, Inject, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatIconModule } from '@angular/material/icon';
import {
  MatAutocompleteModule,
  MatAutocompleteSelectedEvent,
} from '@angular/material/autocomplete';
import { MatChipsModule } from '@angular/material/chips';
import { MatDividerModule } from '@angular/material/divider';
import { FormsModule } from '@angular/forms';
import { ModelsRole, ModelsPolicyRule, RbacService } from '../../generated';
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
    MatChipsModule,
    MatDividerModule,
    FormsModule,
  ],
  template: `
    <h2 mat-dialog-title class="!pt-6">
      <mat-icon class="mr-2 align-middle text-primary">shield_person</mat-icon>
      {{ isEdit ? '编辑角色配置' : '创建新角色' }}
    </h2>
    <mat-dialog-content
      class="flex flex-col gap-6 !pb-4"
      style="min-width: 350px; max-width: 800px;"
    >
      <mat-form-field appearance="outline" class="w-full">
        <mat-label>角色名称</mat-label>
        <input
          matInput
          [(ngModel)]="role.name"
          [disabled]="isEdit"
          placeholder="例如: dns-admin"
          autofocus
        />
        <mat-hint *ngIf="!isEdit">全局唯一的身份标识</mat-hint>
        <mat-error *ngIf="isDuplicate()">角色名称已存在</mat-error>
      </mat-form-field>

      <div class="space-y-4">
        <div class="flex items-center justify-between px-1">
          <span class="text-[10px] font-bold text-outline uppercase tracking-[0.2em]"
            >权限定义列表</span
          >
          <button
            mat-stroked-button
            color="primary"
            class="!rounded-xl scale-90"
            (click)="addRule()"
          >
            <mat-icon class="!w-4 !h-4 !text-[16px]">add</mat-icon>
            添加规则
          </button>
        </div>

        <div class="space-y-4">
          <div
            *ngFor="let rule of role.rules; let i = index"
            class="group relative border border-outline-variant/50 p-4 pt-6 rounded-[24px] bg-surface-container-lowest transition-all hover:border-primary/30 animate-in zoom-in-95 duration-200"
          >
            <!-- Delete Action -->
            <button
              mat-icon-button
              color="warn"
              class="absolute top-1 right-1 !w-8 !h-8 scale-75 sm:opacity-0 group-hover:opacity-100 transition-opacity"
              (click)="removeRule(i)"
              *ngIf="role.rules!.length > 1"
              matTooltip="删除此规则"
            >
              <mat-icon class="!text-[18px]">delete_outline</mat-icon>
            </button>

            <div class="space-y-4">
              <!-- Resource Input -->
              <mat-form-field appearance="outline" class="w-full" subscriptSizing="dynamic">
                <mat-label>资源路径 (Resource)</mat-label>
                <input
                  matInput
                  [(ngModel)]="rule.resource"
                  [matAutocomplete]="autoRes"
                  (input)="onResourceInput(rule)"
                  placeholder="例如: dns/*"
                />
                <mat-autocomplete
                  #autoRes="matAutocomplete"
                  (optionSelected)="onResourceSelected($event, rule)"
                >
                  <mat-option *ngFor="let suggestion of resourceSuggestions()" [value]="suggestion">
                    <mat-icon class="scale-75 opacity-50">category</mat-icon>
                    <span>{{ suggestion }}</span>
                  </mat-option>
                </mat-autocomplete>
              </mat-form-field>

              <!-- Verbs Selection -->
              <mat-form-field appearance="outline" class="w-full" subscriptSizing="dynamic">
                <mat-label>允许操作 (Verbs)</mat-label>
                <mat-chip-grid #chipGridVerb class="!min-h-0">
                  <mat-chip-row
                    *ngFor="let v of rule.verbs"
                    (removed)="removeVerb(rule, v)"
                    class="!bg-secondary-container !text-on-secondary-container !text-[10px] !min-h-[24px]"
                  >
                    {{ v }}
                    <button matChipRemove><mat-icon>cancel</mat-icon></button>
                  </mat-chip-row>
                  <input
                    placeholder="添加..."
                    [matAutocomplete]="autoVerb"
                    [matChipInputFor]="chipGridVerb"
                    #verbInputEl
                    (focus)="onVerbInputFocus(rule)"
                    class="!text-sm"
                  />
                </mat-chip-grid>
                <mat-autocomplete
                  #autoVerb="matAutocomplete"
                  (optionSelected)="onVerbSelected($event, rule, verbInputEl)"
                >
                  <mat-option *ngFor="let verb of verbSuggestions()" [value]="verb">
                    <mat-icon class="scale-75 opacity-50">bolt</mat-icon>
                    <span>{{ verb }}</span>
                  </mat-option>
                </mat-autocomplete>
              </mat-form-field>
            </div>
          </div>
        </div>
      </div>
    </mat-dialog-content>
    <mat-dialog-actions align="end" class="!px-6 !pb-6 !pt-2">
      <button mat-button mat-dialog-close>取消</button>
      <button
        mat-flat-button
        color="primary"
        (click)="confirm()"
        [disabled]="!isValid()"
        class="!ml-2 px-8 rounded-full"
      >
        <mat-icon class="mr-1">check</mat-icon>
        {{ isEdit ? '保存更改' : '立即创建' }}
      </button>
    </mat-dialog-actions>
  `,
})
export class CreateRoleDialogComponent {
  private rbacService = inject(RbacService);
  private dialogRef = inject(MatDialogRef<CreateRoleDialogComponent>);

  isEdit = false;
  role: ModelsRole = {
    name: '',
    rules: [],
  };
  existingNames: string[] = [];

  resourceSuggestions = signal<string[]>([]);
  verbSuggestions = signal<string[]>([]);

  constructor(
    @Inject(MAT_DIALOG_DATA) public data: { role: ModelsRole | null; existingNames?: string[] },
  ) {
    if (data.role) {
      this.isEdit = true;
      this.role = JSON.parse(JSON.stringify(data.role));
    }

    if (!this.role.rules || this.role.rules.length === 0) {
      this.addRule();
    }
    this.existingNames = data.existingNames || [];
  }

  addRule() {
    if (!this.role.rules) this.role.rules = [];
    this.role.rules.push({ resource: '', verbs: [] });
  }

  removeRule(index: number) {
    this.role.rules?.splice(index, 1);
  }

  isDuplicate(): boolean {
    if (this.isEdit) return false;
    return this.existingNames.includes(this.role.name?.trim() || '');
  }

  isValid(): boolean {
    if (!this.role.name || this.isDuplicate()) return false;
    if (!this.role.rules || this.role.rules.length === 0) return false;
    return this.role.rules.every((r) => r.resource && r.verbs && r.verbs.length > 0);
  }

  async onResourceInput(rule: ModelsPolicyRule) {
    const val = (rule.resource || '').trim();
    try {
      const list = await firstValueFrom(this.rbacService.rbacResourcesSuggestGet(val));
      this.resourceSuggestions.set(list || []);
    } catch (e) {
      this.resourceSuggestions.set([]);
    }
  }

  onResourceSelected(event: MatAutocompleteSelectedEvent, rule: ModelsPolicyRule) {
    rule.resource = event.option.viewValue;
    this.updateVerbSuggestions(rule);
  }

  async onVerbInputFocus(rule: ModelsPolicyRule) {
    await this.updateVerbSuggestions(rule);
  }

  async updateVerbSuggestions(rule: ModelsPolicyRule) {
    const resource = rule.resource || '';
    try {
      const list = await firstValueFrom(this.rbacService.rbacVerbsSuggestGet(resource));
      this.verbSuggestions.set(list || []);
    } catch (e) {
      this.verbSuggestions.set([]);
    }
  }

  onVerbSelected(
    event: MatAutocompleteSelectedEvent,
    rule: ModelsPolicyRule,
    inputEl: HTMLInputElement,
  ) {
    if (!rule.verbs) rule.verbs = [];
    const val = event.option.viewValue;
    if (!rule.verbs.includes(val)) {
      rule.verbs.push(val);
    }
    inputEl.value = '';
  }

  removeVerb(rule: ModelsPolicyRule, v: string) {
    if (rule.verbs) {
      rule.verbs = rule.verbs.filter((x) => x !== v);
    }
  }

  confirm() {
    if (this.isValid()) {
      this.dialogRef.close(this.role);
    }
  }
}
