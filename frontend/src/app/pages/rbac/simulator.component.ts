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
import { FormsModule, NgModel } from '@angular/forms';
import {
  RbacService,
  ModelsServiceAccount,
  ModelsResourcePermissions,
  ModelsDiscoverResult,
} from '../../generated';
import { firstValueFrom } from 'rxjs';
import { MatSnackBar } from '@angular/material/snack-bar';

import { DiscoverySelectComponent } from '../../shared/discovery-select.component';
import { DiscoverySuggestInputComponent } from '../../shared/discovery-suggest-input.component';

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
    DiscoverySelectComponent,
    DiscoverySuggestInputComponent,
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
              <!-- ServiceAccount Discovery Select -->
              <div class="md:col-span-2">
                <app-discovery-select
                  code="rbac/serviceaccounts"
                  label="目标服务账号 (ServiceAccount)"
                  placeholder="搜索账号 ID 或名称..."
                  [(ngModel)]="saId"
                  required
                ></app-discovery-select>
              </div>

              <app-discovery-suggest-input
                #resourceModel="ngModel"
                label="资源路径 (Resource)"
                placeholder="例如: rbac/*, dns/example.com, audit/logs"
                [(ngModel)]="resource"
                [rbacSuggestions]="suggestions()"
                [rbacMode]="true"
                (ngModelChange)="onResourceInput()"
                required
              ></app-discovery-suggest-input>

              <mat-form-field appearance="outline" class="w-full">
                <mat-label>动作 (Verb)</mat-label>
                <input
                  matInput
                  [(ngModel)]="verb"
                  [matAutocomplete]="autoVerb"
                  (focus)="onVerbInputFocus()"
                  placeholder="get, list, * ..."
                  required
                  #verbModel="ngModel"
                />
                <mat-autocomplete #autoVerb="matAutocomplete">
                  @for (v of verbSuggestions(); track v) {
                    <mat-option [value]="v">
                      {{ v }}
                    </mat-option>
                  }
                </mat-autocomplete>
                <mat-hint>针对该资源的允许操作</mat-hint>
                @if (verbModel.invalid && verbModel.touched) {
                  <mat-error>此项为必填项</mat-error>
                }
              </mat-form-field>
            </div>

            <div class="mt-8 flex justify-center">
              <button
                mat-fab
                extended
                color="primary"
                class="rounded-2xl! px-12!"
                (click)="simulate()"
                [disabled]="
                  loading() ||
                  !saId() ||
                  !verb() ||
                  !resource() ||
                  resourceModel.invalid ||
                  verbModel.invalid
                "
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
                [class.border-success]="res.allowedAll"
                [class.border-warning]="
                  !res.allowedAll && res.allowedInstances && res.allowedInstances.length > 0
                "
                [class.border-error]="
                  !res.allowedAll && (!res.allowedInstances || res.allowedInstances.length === 0)
                "
              >
                <div class="flex items-center gap-4 mb-8">
                  <div
                    class="w-14 h-14 rounded-2xl flex items-center justify-center shadow-sm"
                    [class.bg-success]="res.allowedAll"
                    [class.text-on-success]="res.allowedAll"
                    [class.bg-warning]="
                      !res.allowedAll && res.allowedInstances && res.allowedInstances.length > 0
                    "
                    [class.text-on-warning]="
                      !res.allowedAll && res.allowedInstances && res.allowedInstances.length > 0
                    "
                    [class.bg-error-container]="
                      !res.allowedAll &&
                      (!res.allowedInstances || res.allowedInstances.length === 0)
                    "
                    [class.text-error]="
                      !res.allowedAll &&
                      (!res.allowedInstances || res.allowedInstances.length === 0)
                    "
                  >
                    @if (res.allowedAll) {
                      <mat-icon class="text-3xl">verified_user</mat-icon>
                    } @else if (res.allowedInstances && res.allowedInstances.length > 0) {
                      <mat-icon class="text-3xl">rule</mat-icon>
                    } @else {
                      <mat-icon class="text-3xl font-bold">gpp_bad</mat-icon>
                    }
                  </div>
                  <div>
                    <h2 class="text-2xl font-black tracking-tight">
                      {{
                        res.allowedAll
                          ? '完全放行'
                          : res.allowedInstances && res.allowedInstances.length > 0
                            ? '局部授权 (按实例)'
                            : '禁止访问'
                      }}
                    </h2>
                    <p class="text-sm opacity-70 font-medium">
                      评估结论基于该账号绑定的所有角色策略
                    </p>
                  </div>
                </div>

                <div class="space-y-8">
                  <!-- Matched Rule Info -->
                  @if (res.matchedRule; as rule) {
                    <div
                      class="flex items-start gap-4 p-5 rounded-2xl bg-secondary-container/20 border border-secondary-container/40 animate-in zoom-in-95 duration-300"
                    >
                      <div
                        class="mt-1 shrink-0 w-10 h-10 rounded-xl bg-secondary/10 text-secondary flex items-center justify-center"
                      >
                        <mat-icon>policy</mat-icon>
                      </div>
                      <div class="flex-1 min-w-0">
                        <div
                          class="text-[11px] font-bold uppercase tracking-widest text-secondary/70"
                        >
                          最优命中规则
                        </div>
                        <div class="mt-2 flex items-center gap-3 flex-wrap">
                          <code
                            class="px-3 py-1 bg-surface-container-high rounded-lg border border-outline-variant font-mono text-sm font-bold text-primary"
                            >{{ rule.resource }}</code
                          >
                          <mat-icon class="text-sm opacity-30">arrow_forward</mat-icon>
                          <div class="flex gap-1.5">
                            @for (v of rule.verbs; track v) {
                              <span
                                class="text-[10px] bg-secondary text-on-secondary px-2 py-0.5 rounded-md uppercase font-black"
                              >
                                {{ v }}
                              </span>
                            }
                          </div>
                        </div>
                        <p class="mt-3 text-xs text-outline leading-relaxed">
                          * 匹配算法已识别此规则为最具体匹配项，足以覆盖您请求的路径。
                        </p>
                      </div>
                    </div>
                  }

                  <div class="grid grid-cols-1 sm:grid-cols-2 gap-6">
                    <!-- Capability: All -->
                    <div
                      class="p-5 rounded-2xl border transition-all duration-300"
                      [class.bg-success-container/20]="res.allowedAll"
                      [class.border-success/30]="res.allowedAll"
                      [class.bg-surface-container-low]="!res.allowedAll"
                      [class.border-outline-variant/50]="!res.allowedAll"
                    >
                      <div class="flex items-center gap-3 mb-3">
                        <mat-icon
                          [class.text-success]="res.allowedAll"
                          [class.text-outline]="!res.allowedAll"
                        >
                          {{ res.allowedAll ? 'check_circle' : 'cancel' }}
                        </mat-icon>
                        <span class="font-bold text-sm">全局授权 (AllowedAll)</span>
                      </div>
                      <p
                        class="text-[12px] leading-relaxed"
                        [class.text-on-surface]="res.allowedAll"
                        [class.text-outline]="!res.allowedAll"
                      >
                        {{
                          res.allowedAll
                            ? '拥有该资源路径下的所有子实例、子操作的完全访问权。'
                            : '未获得全局授权，访问将受限于具体实例规则。'
                        }}
                      </p>
                    </div>

                    <!-- Capability: Specific Instances -->
                    <div
                      class="p-5 rounded-2xl border transition-all duration-300"
                      [class.bg-warning-container/30]="
                        !res.allowedAll && res.allowedInstances && res.allowedInstances.length > 0
                      "
                      [class.border-warning/40]="
                        !res.allowedAll && res.allowedInstances && res.allowedInstances.length > 0
                      "
                      [class.bg-surface-container-low]="
                        res.allowedAll || !res.allowedInstances || res.allowedInstances.length === 0
                      "
                      [class.border-outline-variant/50]="
                        res.allowedAll || !res.allowedInstances || res.allowedInstances.length === 0
                      "
                    >
                      <div class="flex items-center gap-3 mb-3">
                        <mat-icon
                          [class.text-warning]="
                            !res.allowedAll &&
                            res.allowedInstances &&
                            res.allowedInstances.length > 0
                          "
                          [class.text-outline]="
                            res.allowedAll ||
                            !res.allowedInstances ||
                            res.allowedInstances.length === 0
                          "
                        >
                          {{
                            res.allowedInstances && res.allowedInstances.length > 0
                              ? 'check_circle'
                              : 'cancel'
                          }}
                        </mat-icon>
                        <span class="font-bold text-sm">局部授权 (AllowedInstances)</span>
                      </div>
                      <p
                        class="text-[12px] leading-relaxed"
                        [class.text-on-surface]="
                          !res.allowedAll && res.allowedInstances && res.allowedInstances.length > 0
                        "
                        [class.text-outline]="
                          res.allowedAll ||
                          !res.allowedInstances ||
                          res.allowedInstances.length === 0
                        "
                      >
                        {{
                          res.allowedInstances && res.allowedInstances.length > 0
                            ? '仅在下方列出的具体资源实例上拥有操作权限。'
                            : '无特定实例授权或已被全局权限覆盖。'
                        }}
                      </p>
                    </div>
                  </div>

                  @if (res.allowedInstances && res.allowedInstances.length > 0) {
                    <div class="animate-in fade-in slide-in-from-bottom-2 duration-500">
                      <div
                        class="text-[11px] font-bold uppercase tracking-widest text-outline mb-3 ml-1"
                      >
                        允许访问的实例列表
                      </div>
                      <div class="flex flex-wrap gap-2">
                        @for (inst of res.allowedInstances; track inst) {
                          <span
                            class="bg-tertiary-container text-on-tertiary-container px-4 py-1.5 rounded-xl text-xs font-mono font-bold shadow-sm border border-tertiary/10"
                          >
                            {{ inst }}
                          </span>
                        }
                      </div>
                    </div>
                  }
                </div>

                <!-- Explanation -->
                <div
                  class="mt-10 pt-6 border-t border-outline-variant/30 text-[11px] text-outline leading-relaxed italic"
                >
                  <p>评估说明：</p>
                  <ul class="list-disc ml-4 mt-1 space-y-1">
                    <li>如果您输入的路径包含具体实例 (如 dns/example.com)，系统将执行精确匹配。</li>
                    <li>
                      如果您仅输入资源分类 (如 dns)，模拟器将展示该账号在整个分类下的权限概览。
                    </li>
                    <li>所有评估结果均为实时计算，修改角色规则后请重新点击评估。</li>
                  </ul>
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

  saId = signal('');
  verb = signal('');
  resource = signal('');
  suggestions = signal<ModelsDiscoverResult[]>([]);
  verbSuggestions = signal<string[]>([]);
  loading = signal(false);
  result = signal<ModelsResourcePermissions | null>(null);

  ngOnInit() {}

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
          serviceAccountId: this.saId(),
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
