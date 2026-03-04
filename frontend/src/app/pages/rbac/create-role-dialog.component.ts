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
import { MatTooltipModule } from '@angular/material/tooltip';
import { FormsModule } from '@angular/forms';
import { ModelsRole, ModelsPolicyRule, RbacService } from '../../generated';
import { firstValueFrom } from 'rxjs';

import { DiscoverySuggestInputComponent } from '../../shared/discovery-suggest-input.component';

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
    MatTooltipModule,
    FormsModule,
    DiscoverySuggestInputComponent,
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
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        @if (isEdit) {
          <mat-form-field appearance="outline" class="w-full">
            <mat-label>角色 ID (只读)</mat-label>
            <input matInput [value]="role.id" disabled />
          </mat-form-field>
        } @else {
          <div
            class="flex items-center px-4 bg-surface-container rounded-2xl border border-outline-variant/30 h-[56px]"
          >
            <mat-icon class="mr-3 text-outline">info</mat-icon>
            <span class="text-[11px] leading-tight text-outline"
              >角色 ID 将在创建时由系统自动生成 (UUID)</span
            >
          </div>
        }

        <mat-form-field appearance="outline" class="w-full">
          <mat-label>角色名称 (显示名称)</mat-label>
          <input
            matInput
            [(ngModel)]="role.name"
            placeholder="例如: DNS 管理员"
            autofocus
            required
          />
          <mat-hint>描述该角色的用途</mat-hint>
        </mat-form-field>
      </div>

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
          @for (rule of role.rules; track $index; let i = $index) {
            <div
              class="group relative border border-outline-variant/50 p-4 pt-6 rounded-[24px] bg-surface-container-lowest transition-all hover:border-primary/30 animate-in zoom-in-95 duration-200"
            >
              <!-- Delete Action -->
              @if (role.rules!.length > 1) {
                <button
                  mat-icon-button
                  color="warn"
                  class="absolute top-1 right-1 !w-8 !h-8 scale-75 sm:opacity-0 group-hover:opacity-100 transition-opacity icon-button-center"
                  (click)="removeRule(i)"
                  matTooltip="删除此规则"
                >
                  <mat-icon class="!text-[20px]">delete_outline</mat-icon>
                </button>
              }

              <div class="flex flex-col gap-2">
                <!-- Resource Input with Suggestions -->
                <app-discovery-suggest-input
                  label="资源路径 (Resource)"
                  placeholder="例如: rbac/*, dns/example.com, audit/logs"
                  [(ngModel)]="rule.resource"
                  [staticSuggestions]="resourceSuggestions()"
                  staticSuggestionsLabel="资源路径推导"
                  (ngModelChange)="onResourceInput(rule)"
                ></app-discovery-suggest-input>

                <!-- Verbs Selection -->
                <mat-form-field appearance="outline" class="w-full">
                  <mat-label>允许操作 (Verbs)</mat-label>
                  <mat-chip-grid #chipGridVerb>
                    @for (v of rule.verbs; track v) {
                      <mat-chip-row
                        (removed)="removeVerb(rule, v)"
                        class="!bg-secondary-container !text-on-secondary-container"
                      >
                        {{ v }}
                        <button matChipRemove><mat-icon>cancel</mat-icon></button>
                      </mat-chip-row>
                    }
                    <input
                      placeholder="添加..."
                      matInput
                      [matAutocomplete]="autoVerb"
                      [matChipInputFor]="chipGridVerb"
                      #verbInputEl
                      (focus)="onVerbInputFocus(rule)"
                    />
                  </mat-chip-grid>
                  <mat-autocomplete
                    #autoVerb="matAutocomplete"
                    (optionSelected)="onVerbSelected($event, rule, verbInputEl)"
                  >
                    @for (verb of verbSuggestions(); track verb) {
                      <mat-option [value]="verb">
                        <mat-icon class="scale-75 opacity-50">bolt</mat-icon>
                        <span>{{ verb }}</span>
                      </mat-option>
                    }
                  </mat-autocomplete>
                </mat-form-field>
              </div>
            </div>
          }
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
    id: '',
    name: '',
    rules: [],
  };
  existingIDs: string[] = [];

  resourceSuggestions = signal<string[]>([]);
  verbSuggestions = signal<string[]>([]);

  constructor(
    @Inject(MAT_DIALOG_DATA) public data: { role: ModelsRole | null; existingIDs?: string[] },
  ) {
    if (data.role) {
      this.isEdit = true;
      this.role = JSON.parse(JSON.stringify(data.role));
    }

    if (!this.role.rules || this.role.rules.length === 0) {
      this.addRule();
    }
    this.existingIDs = data.existingIDs || [];
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
    if (!this.role.id) return false;
    return this.existingIDs.includes(this.role.id.trim());
  }

  isValid(): boolean {
    if (!this.role.name?.trim()) return false;
    if (this.isEdit && !this.role.id) return false;
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
    // Also trigger verb suggestion update
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
