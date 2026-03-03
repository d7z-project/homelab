import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatChipsModule } from '@angular/material/chips';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import {
  MatAutocompleteModule,
  MatAutocompleteSelectedEvent,
} from '@angular/material/autocomplete';
import { FormsModule } from '@angular/forms';
import { RbacService, ModelsServiceAccount, ModelsResourcePermissions } from '../../generated';
import { firstValueFrom } from 'rxjs';
import { MatSnackBar } from '@angular/material/snack-bar';

@Component({
  selector: 'app-rbac-simulator',
  standalone: true,
  imports: [
    CommonModule,
    MatCardModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatButtonModule,
    MatIconModule,
    MatChipsModule,
    MatProgressSpinnerModule,
    MatAutocompleteModule,
    FormsModule,
  ],
  template: `
    <div class="animate-in fade-in duration-500 pb-20">
      <div class="min-h-[calc(100vh-64px)] bg-surface-container-lowest py-8 px-4 sm:px-8">
        <div class="max-w-4xl mx-auto space-y-8">
          <!-- Header -->
          <div class="flex flex-col gap-1">
            <h1 class="text-3xl font-bold tracking-tight text-on-surface">权限评估模拟器</h1>
            <p class="text-outline text-sm">输入测试参数以预览特定 ServiceAccount 的最终有效权限</p>
          </div>

          <!-- Configuration Card -->
          <div class="bg-surface border border-outline-variant rounded-3xl p-6 sm:p-8 shadow-sm">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
              <!-- ServiceAccount Searchable Autocomplete -->
              <mat-form-field appearance="outline" class="w-full md:col-span-2">
                <mat-label>目标服务账号 (ServiceAccount)</mat-label>
                <input
                  matInput
                  [matAutocomplete]="saAuto"
                  [(ngModel)]="saSearchValue"
                  (input)="onSaSearch($any($event.target).value)"
                  placeholder="搜索账号 ID 或名称..."
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
                    selectedSaID() || '未选择'
                  }}</code></mat-hint
                >
              </mat-form-field>

              <mat-form-field appearance="outline" class="w-full">
                <mat-label>资源路径 (Resource)</mat-label>
                <input
                  matInput
                  [(ngModel)]="resource"
                  [matAutocomplete]="auto"
                  (input)="onResourceInput()"
                  placeholder="dns, rbac, dns/example.com ..."
                />
                <mat-autocomplete #auto="matAutocomplete">
                  @for (suggestion of suggestions(); track suggestion) {
                    <mat-option [value]="suggestion">
                      {{ suggestion }}
                    </mat-option>
                  }
                </mat-autocomplete>
                <mat-hint>例如: dns/example.com</mat-hint>
              </mat-form-field>

              <mat-form-field appearance="outline" class="w-full">
                <mat-label>动作 (Verb)</mat-label>
                <input
                  matInput
                  [(ngModel)]="verb"
                  [matAutocomplete]="autoVerb"
                  (focus)="onVerbInputFocus()"
                  placeholder="get, list, * ..."
                />
                <mat-autocomplete #autoVerb="matAutocomplete">
                  @for (v of verbSuggestions(); track v) {
                    <mat-option [value]="v">
                      {{ v }}
                    </mat-option>
                  }
                </mat-autocomplete>
                <mat-hint>针对该资源的允许操作</mat-hint>
              </mat-form-field>
            </div>

            <div class="mt-8 flex justify-center">
              <button
                mat-fab
                extended
                color="primary"
                class="!rounded-2xl !px-12"
                (click)="simulate()"
                [disabled]="loading() || !selectedSaID() || !verb() || !resource()"
              >
                @if (loading()) {
                  <mat-spinner diameter="20" class="mr-2"></mat-spinner>
                } @else {
                  <mat-icon>play_arrow</mat-icon>
                }
                开始评估
              </button>
            </div>
          </div>

          <!-- Result Card -->
          @if (result(); as res) {
            <div class="animate-in slide-in-from-top-4 duration-500">
              <div
                class="bg-surface border-2 rounded-3xl p-6 sm:p-8 shadow-md transition-all"
                [class.border-primary]="res.allowedAll"
                [class.border-tertiary]="
                  !res.allowedAll && res.allowedInstances && res.allowedInstances.length > 0
                "
                [class.border-outline-variant]="
                  !res.allowedAll && (!res.allowedInstances || res.allowedInstances.length === 0)
                "
              >
                <div class="flex items-center gap-4 mb-6">
                  <div
                    class="w-12 h-12 rounded-2xl flex items-center justify-center"
                    [class.bg-primary]="res.allowedAll"
                    [class.bg-tertiary]="
                      !res.allowedAll && res.allowedInstances && res.allowedInstances.length > 0
                    "
                    [class.bg-error/10]="
                      !res.allowedAll &&
                      (!res.allowedInstances || res.allowedInstances.length === 0)
                    "
                  >
                    @if (res.allowedAll) {
                      <mat-icon class="text-white">verified_user</mat-icon>
                    } @else if (res.allowedInstances && res.allowedInstances.length > 0) {
                      <mat-icon class="text-white">rule</mat-icon>
                    } @else {
                      <mat-icon class="text-error">gpp_bad</mat-icon>
                    }
                  </div>
                  <div>
                    <h2 class="text-xl font-bold">
                      评估结果:
                      {{
                        res.allowedAll
                          ? '完全放行'
                          : res.allowedInstances && res.allowedInstances.length > 0
                            ? '受限放行'
                            : '拒绝访问'
                      }}
                    </h2>
                    <p class="text-sm opacity-60">基于当前所选账号计算得出</p>
                  </div>
                </div>

                <div class="space-y-6">
                  <!-- Matched Rule Info -->
                  @if (res.matchedRule; as rule) {
                    <div
                      class="flex items-start gap-4 p-4 rounded-2xl bg-secondary-container/30 border border-secondary-container/50 animate-in zoom-in-95 duration-300"
                    >
                      <div
                        class="mt-1 flex-shrink-0 w-8 h-8 rounded-full bg-secondary text-on-secondary flex items-center justify-center"
                      >
                        <mat-icon class="text-lg">policy</mat-icon>
                      </div>
                      <div class="flex-1 min-w-0">
                        <div class="text-xs font-bold uppercase tracking-wider text-secondary/80">
                          命中的策略规则
                        </div>
                        <div class="mt-1 flex items-center gap-2 flex-wrap">
                          <code
                            class="px-2 py-0.5 bg-surface rounded border border-outline-variant font-mono text-sm"
                            >{{ rule.resource }}</code
                          >
                          <mat-icon class="text-sm opacity-40">arrow_forward</mat-icon>
                          <div class="flex gap-1">
                            @for (v of rule.verbs; track v) {
                              <span
                                class="text-[10px] bg-secondary text-on-secondary px-1.5 py-0.5 rounded uppercase font-bold"
                              >
                                {{ v }}
                              </span>
                            }
                          </div>
                        </div>
                        <div class="mt-2 text-xs text-outline italic">
                          * 系统判定此规则能够涵盖您的请求。
                        </div>
                      </div>
                    </div>
                  }

                  <!-- Capability: All -->
                  <div class="flex items-start gap-3">
                    <mat-icon
                      class="mt-0.5"
                      [class.text-primary]="res.allowedAll"
                      [class.opacity-20]="!res.allowedAll"
                    >
                      {{ res.allowedAll ? 'check_circle' : 'cancel' }}
                    </mat-icon>
                    <div>
                      <div class="font-bold">全局权限 (AllowedAll)</div>
                      <div class="text-sm opacity-60">允许操作该资源类型下的所有实例</div>
                    </div>
                  </div>

                  <!-- Capability: Specific Instances -->
                  <div class="flex items-start gap-3">
                    <mat-icon
                      class="mt-0.5"
                      [class.text-tertiary]="
                        res.allowedInstances && res.allowedInstances.length > 0
                      "
                      [class.opacity-20]="
                        !res.allowedInstances || res.allowedInstances.length === 0
                      "
                    >
                      {{
                        res.allowedInstances && res.allowedInstances.length > 0
                          ? 'check_circle'
                          : 'cancel'
                      }}
                    </mat-icon>
                    <div class="flex-1">
                      <div class="font-bold">特定实例权限 (AllowedInstances)</div>
                      <div class="text-sm opacity-60 mb-3">仅允许操作下列具体资源</div>

                      @if (res.allowedInstances && res.allowedInstances.length > 0) {
                        <div class="flex flex-wrap gap-2">
                          @for (inst of res.allowedInstances; track inst) {
                            <span
                              class="bg-tertiary-container text-on-tertiary-container px-3 py-1 rounded-full text-xs font-mono font-medium shadow-sm"
                            >
                              {{ inst }}
                            </span>
                          }
                        </div>
                      } @else {
                        <div class="text-xs italic opacity-40">列表为空</div>
                      }
                    </div>
                  </div>
                </div>

                <!-- Explanation -->
                <div
                  class="mt-8 pt-6 border-t border-outline-variant/30 text-xs text-outline leading-relaxed italic"
                >
                  * 提示：如果资源路径输入包含实例（如
                  dns/example.com），系统会自动拆分并优先匹配精确规则。如果仅输入资源类名（如
                  dns），系统将展示该账号在 dns 下的所有能力。
                </div>
              </div>
            </div>
          }
        </div>
      </div>
    </div>
  `,
})
export class RbacSimulatorComponent implements OnInit {
  private rbacService = inject(RbacService);
  private snackBar = inject(MatSnackBar);

  saList = signal<ModelsServiceAccount[]>([]);
  saSearchValue = '';
  filteredSa = signal<ModelsServiceAccount[]>([]);
  selectedSaID = signal('');

  verb = signal('');
  resource = signal('');
  suggestions = signal<string[]>([]);
  verbSuggestions = signal<string[]>([]);
  loading = signal(false);
  result = signal<ModelsResourcePermissions | null>(null);

  ngOnInit() {
    this.onSaSearch(''); // Load initial 50
  }

  displaySaFn(sa: any): string {
    if (typeof sa === 'string') return sa;
    return sa ? sa.name || sa.id : '';
  }

  async onSaSearch(val: string) {
    if (typeof val !== 'string') return;
    try {
      const data = await firstValueFrom(this.rbacService.rbacServiceaccountsGet(1, 50, val));
      this.filteredSa.set(data.items || []);
    } catch (e) {
      this.filteredSa.set([]);
    }
  }

  onSaSelected(event: MatAutocompleteSelectedEvent) {
    const sa = event.option.value as ModelsServiceAccount;
    this.selectedSaID.set(sa.id || '');
    this.saSearchValue = sa.name || sa.id || '';
  }

  async onResourceInput() {
    const val = this.resource().trim();
    try {
      const list = await firstValueFrom(this.rbacService.rbacResourcesSuggestGet(val));
      this.suggestions.set(list || []);
    } catch (e) {
      this.suggestions.set([]);
    }
  }

  async onVerbInputFocus() {
    try {
      const list = await firstValueFrom(this.rbacService.rbacVerbsSuggestGet(this.resource()));
      this.verbSuggestions.set(list || []);
    } catch (e) {
      this.verbSuggestions.set([]);
    }
  }

  async simulate() {
    this.loading.set(true);
    this.result.set(null);
    try {
      const res = await firstValueFrom(
        this.rbacService.rbacSimulatePost({
          serviceAccountId: this.selectedSaID(),
          verb: this.verb(),
          resource: this.resource(),
        }),
      );
      this.result.set(res);
    } catch (err: any) {
      const msg = err.error?.message || '评估失败';
      this.snackBar.open(msg, '关闭', { duration: 3000 });
    } finally {
      this.loading.set(false);
    }
  }
}
