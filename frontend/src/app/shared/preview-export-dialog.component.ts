import { Component, Inject, OnInit, signal, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatTableModule } from '@angular/material/table';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { ReactiveFormsModule, FormBuilder, Validators, FormControl } from '@angular/forms';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { DiscoverySelectComponent } from './discovery-select.component';
import {
  NetworkIpService,
  NetworkSiteService,
  ModelsIPPoolEntry,
  ModelsSitePoolEntry,
} from '../generated';
import { firstValueFrom } from 'rxjs';

export interface PreviewExportData {
  type: 'ip' | 'site';
  rule: string;
  groupIds: string[];
}

@Component({
  selector: 'app-preview-export-dialog',
  standalone: true,
  imports: [
    CommonModule,
    MatDialogModule,
    MatButtonModule,
    MatIconModule,
    MatTableModule,
    MatProgressSpinnerModule,
    MatFormFieldModule,
    MatInputModule,
    ReactiveFormsModule,
    DiscoverySelectComponent,
  ],
  template: `
    <h2
      mat-dialog-title
      class="flex! items-center justify-between p-6! mb-0! bg-surface! text-on-surface"
    >
      <div class="flex items-center gap-4">
        <div
          class="w-12 h-12 rounded-2xl bg-primary-container text-on-primary-container flex items-center justify-center shadow-sm"
        >
          <mat-icon class="w-6! h-6! text-[24px]!">science</mat-icon>
        </div>
        <div class="flex flex-col">
          <span class="text-xl font-bold tracking-tight">表达式预览与计算</span>
          <span class="text-xs text-outline font-medium mt-0.5 opacity-80"
            >实时验证过滤规则匹配结果 (最高展示 50 条)</span
          >
        </div>
      </div>
      <button
        mat-icon-button
        mat-dialog-close
        class="text-outline hover:bg-outline/5 transition-colors"
      >
        <mat-icon>close</mat-icon>
      </button>
    </h2>

    <mat-dialog-content
      class="p-0! max-h-[75vh]! flex flex-col relative bg-surface-container-lowest overflow-hidden"
    >
      <!-- Input Section - Card Style -->
      <div class="px-6 py-6 border-b border-outline-variant/10 bg-surface">
        <div class="flex flex-col gap-6">
          <div class="flex flex-col lg:flex-row gap-6 items-stretch w-full">
            <div class="flex-1 space-y-4">
              <app-discovery-select
                [code]="data.type === 'ip' ? 'network/ip/pools' : 'network/site/pools'"
                [label]="data.type === 'ip' ? '依赖的数据源 (IP 池)' : '依赖的数据源 (域名池)'"
                placeholder="请搜索并添加资产池..."
                [formControl]="form.controls.groupIds"
                [multiple]="true"
                subscriptSizing="dynamic"
                class="w-full search-field-m3"
              ></app-discovery-select>

              <mat-form-field appearance="outline" class="w-full" subscriptSizing="dynamic">
                <mat-label>过滤规则 (Expr 表达式)</mat-label>
                <textarea
                  matInput
                  [formControl]="form.controls.rule"
                  rows="2"
                  class="font-mono text-xs leading-relaxed"
                  placeholder='例如: "cn" in tags && cidr matches "^192\\.168\\."'
                ></textarea>
              </mat-form-field>
            </div>

            <div class="flex items-center justify-center lg:w-[140px] shrink-0">
              <button
                mat-flat-button
                color="primary"
                class="w-full h-full lg:max-h-[110px] min-h-[50px] rounded-2xl! shadow-lg shadow-primary/10 hover:shadow-xl hover:shadow-primary/20 transition-all duration-300 active:scale-95 group"
                [disabled]="loading() || form.invalid"
                (click)="calculate()"
              >
                <div class="flex lg:flex-col items-center justify-center gap-2 px-2 py-1 lg:py-4">
                  @if (loading()) {
                    <mat-spinner diameter="24" color="accent" class="opacity-80!"></mat-spinner>
                    <span class="font-bold tracking-wider text-xs">执行中</span>
                  } @else {
                    <mat-icon
                      class="w-6! h-6! text-[24px]! group-hover:rotate-12 transition-transform duration-300"
                      >play_circle</mat-icon
                    >
                    <span class="font-bold tracking-wider text-xs">测试预览</span>
                  }
                </div>
              </button>
            </div>
          </div>
        </div>
      </div>

      <div class="flex-1 relative overflow-auto min-h-[400px]">
        @if (loading()) {
          <div
            class="absolute inset-0 z-20 flex flex-col items-center justify-center bg-surface-container-lowest/80 backdrop-blur-md animate-in fade-in duration-300"
          >
            <mat-spinner diameter="40" strokeWidth="4"></mat-spinner>
            <p
              class="mt-4 text-sm font-medium text-primary tracking-widest uppercase animate-pulse"
            >
              正在全量计算中...
            </p>
          </div>
        }

        @if (error()) {
          <div
            class="p-12 flex flex-col items-center justify-center text-error animate-in fade-in zoom-in-95 duration-300"
          >
            <div class="w-16 h-16 rounded-full bg-error/10 flex items-center justify-center mb-4">
              <mat-icon class="w-8! h-8! text-[32px]!">error_outline</mat-icon>
            </div>
            <span class="font-bold text-lg mb-2">计算执行失败</span>
            <span class="text-sm opacity-80 text-center max-w-[400px] leading-relaxed">{{
              error()
            }}</span>
          </div>
        } @else {
          <table
            mat-table
            [dataSource]="data.type === 'ip' ? ipResults() : siteResults()"
            class="w-full bg-transparent border-collapse overflow-hidden"
          >
            <ng-container matColumnDef="main">
              <th
                mat-header-cell
                *matHeaderCellDef
                class="bg-surface-container-high/50 text-[11px] font-bold text-outline py-5 px-6 uppercase tracking-wider !border-b-outline-variant/30 text-center"
              >
                {{ data.type === 'ip' ? 'CIDR / IP 地址' : '值 / 匹配规则' }}
              </th>
              <td mat-cell *matCellDef="let element" class="py-4 px-6 !border-b-outline-variant/10">
                <div class="flex items-center gap-3 justify-center">
                  @if (data.type === 'site') {
                    <span
                      class="px-2 py-0.5 rounded text-[10px] font-bold border shrink-0"
                      [class.bg-blue-50]="element.type === 2"
                      [class.text-blue-700]="element.type === 2"
                      [class.border-blue-200]="element.type === 2"
                      [class.bg-purple-50]="element.type === 3"
                      [class.text-purple-700]="element.type === 3"
                      [class.border-purple-200]="element.type === 3"
                      [class.bg-orange-50]="element.type === 0"
                      [class.text-orange-700]="element.type === 0"
                      [class.border-orange-200]="element.type === 0"
                      [class.bg-red-50]="element.type === 1"
                      [class.text-red-700]="element.type === 1"
                      [class.border-red-200]="element.type === 1"
                    >
                      {{
                        element.type === 0
                          ? 'Keyword'
                          : element.type === 1
                            ? 'Regexp'
                            : element.type === 2
                              ? 'Domain'
                              : 'Full'
                      }}
                    </span>
                  }
                  <div class="flex flex-col gap-0.5">
                    <span class="font-bold text-on-surface font-mono">{{
                      element.cidr || element.value
                    }}</span>
                    @if (element.ip) {
                      <span class="text-[10px] text-outline opacity-60 font-mono">{{
                        element.ip
                      }}</span>
                    }
                  </div>
                </div>
              </td>
            </ng-container>

            <ng-container matColumnDef="tags">
              <th
                mat-header-cell
                *matHeaderCellDef
                class="bg-surface-container-high/50 text-[11px] font-bold text-outline py-5 px-6 uppercase tracking-wider !border-b-outline-variant/30 text-center"
              >
                命中的标签
              </th>
              <td mat-cell *matCellDef="let element" class="py-4 px-6 !border-b-outline-variant/10">
                <div class="flex flex-wrap gap-1.5 justify-center">
                  @for (tag of sortTags(element.tags); track tag) {
                    <span
                      class="px-2.5 py-0.5 rounded-full text-[10px] font-bold bg-secondary-container text-on-secondary-container border border-outline-variant/20 shadow-sm"
                    >
                      {{ tag | uppercase }}
                    </span>
                  }
                  @if (!element.tags?.length) {
                    <span class="text-xs text-outline opacity-30 italic">无标签</span>
                  }
                </div>
              </td>
            </ng-container>

            <tr mat-header-row *matHeaderRowDef="['main', 'tags']; sticky: true" class="h-14"></tr>
            <tr
              mat-row
              *matRowDef="let row; columns: ['main', 'tags']"
              class="hover:bg-primary/5 transition-colors h-16 cursor-default"
            ></tr>

            <tr *matNoDataRow>
              <td colspan="2" class="py-24 text-center">
                <div
                  class="flex flex-col items-center animate-in fade-in slide-in-from-top-4 duration-500"
                >
                  <div
                    class="w-20 h-20 rounded-full bg-surface-container-high flex items-center justify-center text-outline/30 mb-4"
                  >
                    <mat-icon class="w-10! h-10! text-[40px]!">manage_search</mat-icon>
                  </div>
                  <p class="text-on-surface text-sm font-bold tracking-wide">暂无匹配数据</p>
                  <p class="text-[11px] text-outline opacity-60 mt-2 max-w-[240px] leading-relaxed">
                    请确认数据池内含有数据，或尝试修改过滤规则后重新执行计算。
                  </p>
                </div>
              </td>
            </tr>
          </table>
        }
      </div>
    </mat-dialog-content>

    <div
      class="px-6 py-4 bg-surface border-t border-outline-variant/10 flex justify-between items-center text-[10px] text-outline font-medium"
    >
      <div class="flex items-center gap-2 uppercase tracking-tight opacity-70">
        <mat-icon class="w-4! h-4! text-[14px]!">bolt</mat-icon>
        <span>REAL-TIME PREVIEW ENGINE (Limit: 50 Records)</span>
      </div>
      <button
        mat-flat-button
        color="secondary"
        mat-dialog-close
        class="rounded-xl! px-8 h-10! text-xs font-bold shadow-none"
      >
        关闭窗口
      </button>
    </div>
  `,
  styles: [
    `
      :host {
        display: block;
      }
      .search-field-m3 {
        display: block;
        margin-bottom: 1.5rem;
        ::ng-deep .mdc-text-field--filled {
          background-color: var(--mat-sys-surface) !important;
          border-radius: 12px !important;
        }
        ::ng-deep .mdc-line-ripple {
          display: none;
        }
      }
      textarea {
        scrollbar-width: thin;
        scrollbar-color: var(--mat-sys-outline-variant) transparent;
      }
    `,
  ],
})
export class PreviewExportDialogComponent implements OnInit {
  private ipService = inject(NetworkIpService);
  private siteService = inject(NetworkSiteService);
  private fb = inject(FormBuilder);
  public data = inject<PreviewExportData>(MAT_DIALOG_DATA);

  loading = signal(false);
  error = signal<string | null>(null);
  ipResults = signal<ModelsIPPoolEntry[]>([]);
  siteResults = signal<ModelsSitePoolEntry[]>([]);

  form = this.fb.group({
    rule: [this.data.rule || 'true', Validators.required],
    groupIds: [this.data.groupIds || [], Validators.required],
  });

  constructor(public dialogRef: MatDialogRef<PreviewExportDialogComponent>) {}

  ngOnInit() {
    if (this.data.rule && this.data.groupIds?.length > 0) {
      this.calculate();
    }
  }

  async calculate() {
    if (this.form.invalid) return;
    this.loading.set(true);
    this.error.set(null);
    const val = this.form.value as { rule: string; groupIds: string[] };
    try {
      if (this.data.type === 'ip') {
        const res = await firstValueFrom(
          this.ipService.networkIpExportsPreviewPost({
            rule: val.rule!,
            groupIds: val.groupIds!,
          }),
        );
        this.ipResults.set(res || []);
      } else {
        const res = await firstValueFrom(
          this.siteService.networkSiteExportsPreviewPost({
            rule: val.rule!,
            groupIds: val.groupIds!,
          }),
        );
        this.siteResults.set(res || []);
      }
    } catch (err: any) {
      this.error.set(err.error?.message || err.message || '计算遇到未知错误');
    } finally {
      this.loading.set(false);
    }
  }

  sortTags(tags: string[] | undefined): string[] {
    if (!tags) return [];
    return [...tags].sort((a, b) => {
      const a_is_system = a.startsWith('_');
      const b_is_system = b.startsWith('_');
      if (a_is_system && !b_is_system) return -1;
      if (!a_is_system && b_is_system) return 1;
      return a.localeCompare(b);
    });
  }
}
