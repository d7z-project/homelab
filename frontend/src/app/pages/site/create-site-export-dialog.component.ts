import { Component, inject, signal, Inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, Validators } from '@angular/forms';
import {
  MatDialogRef,
  MatDialogModule,
  MAT_DIALOG_DATA,
  MatDialog,
} from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatIconModule } from '@angular/material/icon';
import { MatExpansionModule } from '@angular/material/expansion';
import { DiscoverySelectComponent } from '../../shared/discovery-select.component';
import { NetworkSiteService, ModelsSiteExport } from '../../generated';

@Component({
  selector: 'app-create-site-export-dialog',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    MatIconModule,
    MatExpansionModule,
    DiscoverySelectComponent,
  ],
  template: `
    <h2 mat-dialog-title>{{ data.export ? '编辑域名导出配置' : '新建域名导出配置' }}</h2>
    <mat-dialog-content>
      <form [formGroup]="form" class="flex flex-col gap-4 pt-2">
        <mat-form-field appearance="outline">
          <mat-label>配置名称</mat-label>
          <input matInput formControlName="name" required />
        </mat-form-field>

        <mat-form-field appearance="outline">
          <mat-label>描述</mat-label>
          <textarea matInput formControlName="description" rows="2"></textarea>
        </mat-form-field>

        <mat-form-field appearance="outline">
          <mat-label>过滤规则 (go-expr)</mat-label>
          <textarea
            matInput
            formControlName="rule"
            required
            rows="3"
            class="font-mono text-sm"
          ></textarea>
        </mat-form-field>

        <mat-expansion-panel
          class="bg-surface-container-low! rounded-xl! overflow-hidden border border-outline-variant/30"
        >
          <mat-expansion-panel-header class="h-10!">
            <mat-panel-title class="flex items-center gap-2 text-xs font-bold text-primary">
              <mat-icon class="w-4! h-4! text-[16px]! flex items-center justify-center"
                >help_outline</mat-icon
              >
              过滤规则编写指南 (go-expr)
            </mat-panel-title>
          </mat-expansion-panel-header>
          <div class="text-[11px] space-y-3 text-outline leading-relaxed pb-3 px-1">
            <p>
              本系统基于
              <a
                href="https://github.com/expr-lang/expr"
                target="_blank"
                class="text-primary hover:underline"
                >go-expr</a
              >
              引擎进行动态过滤。规则必须返回 <b>true</b> 以保留条目。
            </p>

            <div class="space-y-1.5">
              <div class="font-bold text-on-surface flex items-center gap-1">
                <mat-icon class="w-3! h-3! text-[12px]!">variable</mat-icon>
                可用变量
              </div>
              <div class="grid grid-cols-1 gap-1 pl-4">
                <div><code>tags</code>: 标签列表 (如 <code>["cn", "sync"]</code>)</div>
                <div><code>domain</code>: 规则值 (如 <code>"google.com"</code>)</div>
                <div><code>type</code>: 类型 (0:kw, 1:re, 2:dom, 3:full)</div>
              </div>
            </div>

            <div class="space-y-1.5">
              <div class="font-bold text-on-surface flex items-center gap-1">
                <mat-icon class="w-3! h-3! text-[12px]!">lightbulb</mat-icon>
                常用示例
              </div>
              <div class="space-y-2 pl-4">
                <div class="p-2 bg-surface-container border border-outline-variant/20 rounded-lg">
                  <div class="text-on-surface-variant font-medium mb-1">标签过滤</div>
                  <code>"cn" in tags</code>
                </div>
                <div class="p-2 bg-surface-container border border-outline-variant/20 rounded-lg">
                  <div class="text-on-surface-variant font-medium mb-1">
                    复合逻辑 (仅限 Domain 类型的中国域名)
                  </div>
                  <code>type == 2 && "cn" in tags</code>
                </div>
                <div class="p-2 bg-surface-container border border-outline-variant/20 rounded-lg">
                  <div class="text-on-surface-variant font-medium mb-1">正则匹配</div>
                  <code>domain matches "google"</code>
                </div>
              </div>
            </div>
          </div>
        </mat-expansion-panel>

        <app-discovery-select
          code="network/site/pools"
          label="依赖的数据池"
          placeholder="搜索域名池..."
          formControlName="groupIds"
          [multiple]="true"
          required
        ></app-discovery-select>
      </form>
    </mat-dialog-content>
    <mat-dialog-actions align="end">
      <button mat-button mat-dialog-close>取消</button>
      <button
        mat-flat-button
        color="primary"
        [disabled]="form.invalid || loading()"
        (click)="submit()"
      >
        {{ data.export ? '保存' : '创建' }}
      </button>
    </mat-dialog-actions>
  `,
})
export class CreateSiteExportDialogComponent {
  private fb = inject(FormBuilder);
  private siteService = inject(NetworkSiteService);
  private dialogRef = inject(MatDialogRef<CreateSiteExportDialogComponent>);
  private snackBar = inject(MatSnackBar);
  private dialog = inject(MatDialog);

  loading = signal(false);
  form: any;

  constructor(@Inject(MAT_DIALOG_DATA) public data: { export?: ModelsSiteExport }) {
    this.form = this.fb.group({
      name: [this.data.export?.meta?.name || '', Validators.required],
      description: [this.data.export?.meta?.description || ''],
      rule: [this.data.export?.meta?.rule || '"cn" in tags', Validators.required],
      groupIds: [this.data.export?.meta?.groupIds || ([] as string[]), Validators.required],
    });
  }

  submit() {
    if (this.form.invalid) return;
    this.loading.set(true);
    const val = this.form.value;

    const exportData: ModelsSiteExport = {
      id: this.data.export?.id,
      generation: this.data.export?.generation || 0,
      meta: {
        name: val.name!,
        description: val.description || undefined,
        rule: val.rule!,
        groupIds: val.groupIds || [],
      },
    };

    const obs = this.data.export?.id
      ? this.siteService.networkSiteExportsIdPut(this.data.export.id, exportData)
      : this.siteService.networkSiteExportsPost(exportData);

    obs.subscribe({
      next: () => {
        this.snackBar.open(this.data.export ? '更新成功' : '创建成功', '关闭', { duration: 3000 });
        this.dialogRef.close(true);
      },
      error: (err: any) => {
        this.loading.set(false);
        this.snackBar.open(`操作失败: ${err.error?.message || err.message}`, '关闭', {
          duration: 3000,
        });
      },
    });
  }
}
