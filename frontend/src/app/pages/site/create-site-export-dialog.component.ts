import { Component, inject, signal, Inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, Validators } from '@angular/forms';
import { MatDialogRef, MatDialogModule, MAT_DIALOG_DATA, MatDialog } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatIconModule } from '@angular/material/icon';
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
          <mat-hint>可用变量: tags ([]string), domain (string), type (uint8)</mat-hint>
        </mat-form-field>

        <app-discovery-select
          code="network/site/pools"
          label="依赖的数据池"
          placeholder="搜索地址池..."
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
      name: [this.data.export?.name || '', Validators.required],
      description: [this.data.export?.description || ''],
      rule: [this.data.export?.rule || '"cn" in tags', Validators.required],
      groupIds: [this.data.export?.groupIds || ([] as string[]), Validators.required],
    });
  }


  submit() {
    if (this.form.invalid) return;
    this.loading.set(true);
    const val = this.form.value;

    const exportData: ModelsSiteExport = {
      ...this.data.export,
      name: val.name!,
      description: val.description || undefined,
      rule: val.rule!,
      groupIds: val.groupIds || [],
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
