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
import { PreviewExportDialogComponent } from '../../shared/preview-export-dialog.component';
import { NetworkIpService, ModelsIPExport } from '../../generated';

@Component({
  selector: 'app-create-export-dialog',
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
    <h2 mat-dialog-title>{{ data.export ? '编辑导出配置' : '新建导出配置' }}</h2>
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

        <div class="flex items-start gap-4">
          <mat-form-field appearance="outline" class="flex-1">
            <mat-label>过滤规则 (go-expr)</mat-label>
            <textarea
              matInput
              formControlName="rule"
              required
              rows="3"
              class="font-mono text-sm"
            ></textarea>
            <mat-hint
              >可用变量: tags ([]string), cidr (string), ip (string). 示例:
              <code>"cn" in tags || cidr == "8.8.8.8/32"</code></mat-hint
            >
          </mat-form-field>
          <button mat-flat-button color="tertiary" type="button" class="mt-2" [disabled]="!form.get('rule')?.value || !form.get('groupIds')?.value?.length" (click)="previewRule()">
            <mat-icon>science</mat-icon>
            计算预览
          </button>
        </div>

        <app-discovery-select
          code="network/ip/pools"
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
export class CreateExportDialogComponent {
  private fb = inject(FormBuilder);
  private ipService = inject(NetworkIpService);
  private dialogRef = inject(MatDialogRef<CreateExportDialogComponent>);
  private snackBar = inject(MatSnackBar);
  private dialog = inject(MatDialog);

  loading = signal(false);
  form: any;

  constructor(@Inject(MAT_DIALOG_DATA) public data: { export?: ModelsIPExport }) {
    this.form = this.fb.group({
      name: [this.data.export?.name || '', Validators.required],
      description: [this.data.export?.description || ''],
      rule: [this.data.export?.rule || '"cn" in tags', Validators.required],
      groupIds: [this.data.export?.groupIds || ([] as string[]), Validators.required],
    });
  }

  previewRule() {
    this.dialog.open(PreviewExportDialogComponent, {
      width: '700px',
      data: {
        type: 'ip',
        rule: this.form.get('rule')?.value,
        groupIds: this.form.get('groupIds')?.value
      }
    });
  }

  submit() {
    if (this.form.invalid) return;
    this.loading.set(true);
    const val = this.form.value;

    const exportData: ModelsIPExport = {
      ...this.data.export,
      name: val.name!,
      description: val.description || undefined,
      rule: val.rule!,
      groupIds: val.groupIds || [],
    };

    const obs = this.data.export?.id
      ? this.ipService.networkIpExportsIdPut(this.data.export.id, exportData)
      : this.ipService.networkIpExportsPost(exportData);

    obs.subscribe({
      next: () => {
        this.snackBar.open(this.data.export ? '更新成功' : '创建成功', '关闭', { duration: 3000 });
        this.dialogRef.close(true);
      },
      error: (err) => {
        this.loading.set(false);
        this.snackBar.open(`操作失败: ${err.error?.message || err.message}`, '关闭', {
          duration: 3000,
        });
      },
    });
  }
}
