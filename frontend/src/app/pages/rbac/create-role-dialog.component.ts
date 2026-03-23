import {
  Component,
  Inject,
  inject,
  signal,
  ViewChildren,
  QueryList,
  ChangeDetectorRef,
  AfterViewInit,
} from '@angular/core';
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
import { FormsModule, NgModel } from '@angular/forms';
import { ModelsRole, ModelsPolicyRule, RbacService, ModelsDiscoverResult } from '../../generated';
import { firstValueFrom } from 'rxjs';

import { DiscoverySuggestInputComponent } from '../../shared/discovery-suggest-input.component';

interface RuleWithUI extends ModelsPolicyRule {
  resourceSuggestions: ModelsDiscoverResult[];
  verbSuggestions: string[];
  loading: boolean;
}

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
    <h2 mat-dialog-title class="pt-6!">
      <mat-icon class="mr-2 align-middle text-primary">shield_person</mat-icon>
      {{ isEdit ? '编辑角色配置' : '创建新角色' }}
    </h2>
    <mat-dialog-content
      class="flex flex-col gap-6 pb-4!"
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
            [(ngModel)]="role.meta!.name"
            (ngModelChange)="onRuleChange()"
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
            class="rounded-xl! scale-90"
            (click)="addRule()"
          >
            <mat-icon class="w-4! h-4! text-[16px]!">add</mat-icon>
            添加规则
          </button>
        </div>

        <div class="space-y-4">
          @for (rule of rules; track $index; let i = $index) {
            <div
              class="group relative border border-outline-variant/50 p-4 pt-6 rounded-[24px] bg-surface-container-lowest transition-all hover:border-primary/30 animate-in zoom-in-95 duration-200"
            >
              <!-- Delete Action -->
              @if (rules.length > 1) {
                <button
                  mat-icon-button
                  color="warn"
                  class="absolute top-1 right-1 w-8! h-8! scale-75 sm:opacity-0 group-hover:opacity-100 transition-opacity icon-button-center"
                  (click)="removeRule(i)"
                  matTooltip="删除此规则"
                >
                  <mat-icon class="text-[20px]!">delete_outline</mat-icon>
                </button>
              }

              <div class="flex flex-col gap-2">
                <!-- Resource Input with Suggestions -->
                <app-discovery-suggest-input
                  #resourceModel="ngModel"
                  label="资源路径 (Resource)"
                  placeholder="例如: rbac/*, dns/example.com, audit/logs"
                  [(ngModel)]="rule.resource"
                  [rbacSuggestions]="rule.resourceSuggestions"
                  [loading]="rule.loading"
                  [rbacMode]="true"
                  (ngModelChange)="onResourceInput(rule)"
                  required
                ></app-discovery-suggest-input>

                <!-- Verbs Selection -->
                <mat-form-field appearance="outline" class="w-full">
                  <mat-label>允许操作 (Verbs)</mat-label>
                  <mat-chip-grid #chipGridVerb>
                    @for (v of rule.verbs; track v) {
                      <mat-chip-row
                        (removed)="removeVerb(rule, v)"
                        class="bg-secondary-container! text-on-secondary-container!"
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
                    @for (verb of rule.verbSuggestions; track verb) {
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
    <mat-dialog-actions align="end" class="px-6! pb-6! pt-2!">
      <button mat-button mat-dialog-close>取消</button>
      <button
        mat-flat-button
        color="primary"
        (click)="confirm()"
        [disabled]="!formValid()"
        class="ml-2! px-8 rounded-full"
      >
        <mat-icon class="mr-1">check</mat-icon>
        {{ isEdit ? '保存更改' : '立即创建' }}
      </button>
    </mat-dialog-actions>
  `,
})
export class CreateRoleDialogComponent implements AfterViewInit {
  private rbacService = inject(RbacService);
  private dialogRef = inject(MatDialogRef<CreateRoleDialogComponent>);
  private cdr = inject(ChangeDetectorRef);

  @ViewChildren('resourceModel') resourceModels!: QueryList<NgModel>;

  isEdit = false;
  role: ModelsRole = {
    id: '',
    meta: {
      name: '',
      rules: [],
    },
    status: {},
  };
  rules: RuleWithUI[] = [];
  existingIDs: string[] = [];
  formValid = signal(false);

  constructor(
    @Inject(MAT_DIALOG_DATA) public data: { role: ModelsRole | null; existingIDs?: string[] },
  ) {
    if (data.role) {
      this.isEdit = true;
      this.role = JSON.parse(JSON.stringify(data.role));
      this.rules = (this.role.meta?.rules || []).map((r) => ({
        ...r,
        resourceSuggestions: [],
        verbSuggestions: [],
        loading: false,
      }));
    }

    if (this.rules.length === 0) {
      this.addRule();
    }
    this.existingIDs = data.existingIDs || [];
  }

  ngAfterViewInit() {
    // For existing rules, we need to trigger suggestion loading
    // Use setTimeout to ensure we are completely out of the current change detection cycle
    if (this.isEdit) {
      setTimeout(() => {
        this.rules.forEach((r) => {
          if (r.resource) {
            this.onResourceInput(r);
          }
        });
        this.onRuleChange();
      });
    } else {
      setTimeout(() => this.onRuleChange());
    }
  }

  addRule() {
    this.rules.push({
      resource: '',
      verbs: [],
      resourceSuggestions: [],
      verbSuggestions: [],
      loading: false,
    });
    setTimeout(() => this.onRuleChange());
  }

  removeRule(index: number) {
    this.rules.splice(index, 1);
    setTimeout(() => this.onRuleChange());
  }

  onRuleChange() {
    this.formValid.set(this.calculateValidity());
    this.cdr.markForCheck();
  }

  calculateValidity(): boolean {
    if (!this.role.meta?.name?.trim()) return false;
    if (this.isEdit && !this.role.id) return false;
    if (this.rules.length === 0) return false;

    // Check if any resource input is invalid (e.g. notFinal)
    if (this.resourceModels && this.resourceModels.some((m) => !!m.invalid)) {
      return false;
    }

    return this.rules.every((r) => {
      if (!r.resource || !r.verbs || r.verbs.length === 0) return false;
      // Extra safety check for trailing slash or common prefixes
      if (r.resource.endsWith('/')) return false;
      return true;
    });
  }

  async onResourceInput(rule: RuleWithUI) {
    const val = (rule.resource || '').trim();
    // Wrap state changes in microtask or timeout to avoid sync detection issues
    Promise.resolve().then(() => {
      rule.loading = true;
      this.cdr.markForCheck();
    });

    try {
      const list = await firstValueFrom(this.rbacService.rbacResourcesSuggestGet(val));
      // Use setTimeout to ensure the data update happens in a fresh cycle
      // resolving the "Expression has changed after it was checked" error
      setTimeout(() => {
        rule.resourceSuggestions = list || [];
        rule.loading = false;
        this.cdr.markForCheck();
        // Also trigger verb suggestion update
        this.updateVerbSuggestions(rule);
        this.onRuleChange();
      });
    } catch (e) {
      setTimeout(() => {
        rule.resourceSuggestions = [];
        rule.loading = false;
        this.cdr.markForCheck();
        this.onRuleChange();
      });
    }
  }

  async onVerbInputFocus(rule: RuleWithUI) {
    await this.updateVerbSuggestions(rule);
  }

  async updateVerbSuggestions(rule: RuleWithUI) {
    const resource = rule.resource || '';
    try {
      const list = await firstValueFrom(this.rbacService.rbacVerbsSuggestGet(resource));
      setTimeout(() => {
        rule.verbSuggestions = list || [];
        this.cdr.markForCheck();
        this.onRuleChange();
      });
    } catch (e) {
      setTimeout(() => {
        rule.verbSuggestions = [];
        this.cdr.markForCheck();
        this.onRuleChange();
      });
    }
  }

  onVerbSelected(event: MatAutocompleteSelectedEvent, rule: RuleWithUI, inputEl: HTMLInputElement) {
    if (!rule.verbs) rule.verbs = [];
    const val = event.option.viewValue;
    if (!rule.verbs.includes(val)) {
      rule.verbs.push(val);
    }
    inputEl.value = '';
    this.onRuleChange();
  }

  removeVerb(rule: RuleWithUI, v: string) {
    if (rule.verbs) {
      rule.verbs = rule.verbs.filter((x) => x !== v);
    }
    this.onRuleChange();
  }

  confirm() {
    if (this.calculateValidity()) {
      // Sync rules back to role model
      this.role.meta!.rules = this.rules.map(({ resource, verbs }) => ({ resource, verbs }));
      this.dialogRef.close(this.role);
    }
  }
}
