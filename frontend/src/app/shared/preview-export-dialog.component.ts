import { Component, Inject, OnInit, signal, computed, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatTableModule } from '@angular/material/table';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { NetworkIpService, NetworkSiteService, ModelsIPPoolEntry, ModelsSitePoolEntry } from '../generated';
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
  ],
  template: `
    <h2 mat-dialog-title class="!flex items-center justify-between">
      <div class="flex items-center gap-2">
        <mat-icon color="primary">science</mat-icon>
        <span>表达式计算结果 (前 50 条)</span>
      </div>
      <button mat-icon-button mat-dialog-close><mat-icon>close</mat-icon></button>
    </h2>

    <mat-dialog-content class="!p-0 min-h-[400px] flex flex-col relative bg-surface-container-lowest">
      @if (loading()) {
        <div class="absolute inset-0 z-10 flex flex-col items-center justify-center bg-surface/50 backdrop-blur-sm">
          <mat-spinner diameter="40"></mat-spinner>
          <span class="mt-4 text-sm text-outline font-medium">正在计算中...</span>
        </div>
      }

      @if (!loading() && error()) {
        <div class="p-8 flex flex-col items-center justify-center text-error">
          <mat-icon class="!w-12 !h-12 !text-[48px] mb-4 opacity-80">error_outline</mat-icon>
          <span class="font-bold mb-2">计算失败</span>
          <span class="text-sm opacity-80 text-center max-w-[80%]">{{ error() }}</span>
        </div>
      }

      @if (!loading() && !error()) {
        @if (data.type === 'ip') {
          <table mat-table [dataSource]="ipResults()" class="w-full bg-transparent">
            <ng-container matColumnDef="cidr">
              <th mat-header-cell *matHeaderCellDef class="bg-surface-container-low font-bold">CIDR / IP</th>
              <td mat-cell *matCellDef="let element" class="py-3">
                <span class="font-mono text-sm font-bold text-primary">{{ element.cidr }}</span>
              </td>
            </ng-container>
            <ng-container matColumnDef="tags">
              <th mat-header-cell *matHeaderCellDef class="bg-surface-container-low font-bold">匹配标签</th>
              <td mat-cell *matCellDef="let element" class="py-3">
                <div class="flex flex-wrap gap-1.5">
                  @for (tag of element.tags; track tag) {
                    <span class="px-2 py-0.5 rounded-md text-[10px] font-bold border"
                          [class.bg-surface-container]="tag.startsWith('_')"
                          [class.text-outline]="tag.startsWith('_')"
                          [class.border-outline-variant]="tag.startsWith('_')"
                          [class.bg-primary-container]="!tag.startsWith('_')"
                          [class.text-on-primary-container]="!tag.startsWith('_')"
                          [class.border-primary]="!tag.startsWith('_')">
                      {{ tag }}
                    </span>
                  }
                  @if (!element.tags || element.tags.length === 0) {
                    <span class="text-xs text-outline italic">无标签</span>
                  }
                </div>
              </td>
            </ng-container>
            <tr mat-header-row *matHeaderRowDef="['cidr', 'tags']; sticky: true"></tr>
            <tr mat-row *matRowDef="let row; columns: ['cidr', 'tags']" class="hover:bg-surface-container transition-colors"></tr>
          </table>
        } @else {
          <table mat-table [dataSource]="siteResults()" class="w-full bg-transparent">
            <ng-container matColumnDef="value">
              <th mat-header-cell *matHeaderCellDef class="bg-surface-container-low font-bold">域名 / 规则</th>
              <td mat-cell *matCellDef="let element" class="py-3">
                <div class="flex items-center gap-2">
                  <span class="px-1.5 py-0.5 rounded text-[10px] font-bold border"
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
                    [class.border-red-200]="element.type === 1">
                    {{ element.type === 0 ? 'Keyword' : element.type === 1 ? 'Regexp' : element.type === 2 ? 'Domain' : 'Full' }}
                  </span>
                  <span class="font-mono text-sm font-bold text-primary">{{ element.value }}</span>
                </div>
              </td>
            </ng-container>
            <ng-container matColumnDef="tags">
              <th mat-header-cell *matHeaderCellDef class="bg-surface-container-low font-bold">匹配标签</th>
              <td mat-cell *matCellDef="let element" class="py-3">
                <div class="flex flex-wrap gap-1.5">
                  @for (tag of element.tags; track tag) {
                    <span class="px-2 py-0.5 rounded-md text-[10px] font-bold border"
                          [class.bg-surface-container]="tag.startsWith('_')"
                          [class.text-outline]="tag.startsWith('_')"
                          [class.border-outline-variant]="tag.startsWith('_')"
                          [class.bg-primary-container]="!tag.startsWith('_')"
                          [class.text-on-primary-container]="!tag.startsWith('_')"
                          [class.border-primary]="!tag.startsWith('_')">
                      {{ tag }}
                    </span>
                  }
                  @if (!element.tags || element.tags.length === 0) {
                    <span class="text-xs text-outline italic">无标签</span>
                  }
                </div>
              </td>
            </ng-container>
            <tr mat-header-row *matHeaderRowDef="['value', 'tags']; sticky: true"></tr>
            <tr mat-row *matRowDef="let row; columns: ['value', 'tags']" class="hover:bg-surface-container transition-colors"></tr>
          </table>
        }

        @if ((data.type === 'ip' && ipResults().length === 0) || (data.type === 'site' && siteResults().length === 0)) {
          <div class="flex-1 flex flex-col items-center justify-center p-12 text-outline/40 italic">
            <mat-icon class="!w-12 !h-12 !text-[48px] mb-4 opacity-20">find_in_page</mat-icon>
            <span>无匹配数据</span>
          </div>
        }
      }
    </mat-dialog-content>

    <div mat-dialog-actions align="end" class="!px-6 !py-4 border-t border-outline-variant/30">
      <div class="flex-1 text-xs text-outline text-left italic">
        仅显示前 50 条匹配结果
      </div>
      <button mat-button mat-dialog-close class="px-6 !rounded-xl">关闭</button>
    </div>
  `,
  styles: [`
    :host { display: block; }
  `]
})
export class PreviewExportDialogComponent implements OnInit {
  private ipService = inject(NetworkIpService);
  private siteService = inject(NetworkSiteService);

  loading = signal(true);
  error = signal<string | null>(null);
  ipResults = signal<ModelsIPPoolEntry[]>([]);
  siteResults = signal<ModelsSitePoolEntry[]>([]);

  constructor(
    public dialogRef: MatDialogRef<PreviewExportDialogComponent>,
    @Inject(MAT_DIALOG_DATA) public data: PreviewExportData
  ) {}

  ngOnInit() {
    this.calculate();
  }

  async calculate() {
    this.loading.set(true);
    this.error.set(null);
    try {
      if (this.data.type === 'ip') {
        const res = await firstValueFrom(this.ipService.networkIpExportsPreviewPost({
          rule: this.data.rule,
          groupIds: this.data.groupIds
        }));
        this.ipResults.set(res || []);
      } else {
        const res = await firstValueFrom(this.siteService.networkSiteExportsPreviewPost({
          rule: this.data.rule,
          groupIds: this.data.groupIds
        }));
        this.siteResults.set(res || []);
      }
    } catch (err: any) {
      this.error.set(err.error?.message || err.message || '计算遇到未知错误');
    } finally {
      this.loading.set(false);
    }
  }
}
