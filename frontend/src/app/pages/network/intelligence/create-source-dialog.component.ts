import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, Validators } from '@angular/forms';
import { MatDialogRef, MatDialogModule } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatSnackBar } from '@angular/material/snack-bar';
import { NetworkIntelligenceService } from '../../../generated';

@Component({
  selector: 'app-create-source-dialog',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
  ],
  template: `
    <h2 mat-dialog-title>配置情报数据源</h2>
    <mat-dialog-content>
      <form [formGroup]="form" class="flex flex-col gap-4 pt-2">
        <mat-form-field appearance="outline">
          <mat-label>名称</mat-label>
          <input matInput formControlName="name" required placeholder="例如：MaxMind ASN 官方源" />
        </mat-form-field>

        <mat-form-field appearance="outline">
          <mat-label>数据库类型</mat-label>
          <mat-select formControlName="type" required>
            <mat-option value="asn">ASN 数据库</mat-option>
            <mat-option value="city">城市地理位置 (City)</mat-option>
            <mat-option value="country">国家地理位置 (Country)</mat-option>
          </mat-select>
        </mat-form-field>

        <mat-form-field appearance="outline">
          <mat-label>下载直链 (URL)</mat-label>
          <input matInput formControlName="url" required placeholder="https://..." />
          <mat-hint>必须是可直接下载的 .mmdb 原始文件链接</mat-hint>
        </mat-form-field>
      </form>
    </mat-dialog-content>
    <mat-dialog-actions align="end">
      <button mat-button mat-dialog-close>取消</button>
      <button mat-flat-button color="primary" [disabled]="form.invalid || loading" (click)="submit()">
        保存配置
      </button>
    </mat-dialog-actions>
  `,
})
export class CreateSourceDialogComponent {
  private fb = inject(FormBuilder);
  private intService = inject(NetworkIntelligenceService);
  private dialogRef = inject(MatDialogRef<CreateSourceDialogComponent>);
  private snackBar = inject(MatSnackBar);

  loading = false;

  form = this.fb.group({
    name: ['', Validators.required],
    type: ['asn', Validators.required],
    url: ['', [Validators.required, Validators.pattern(/^https?:\/\/.+$/)]],
  });

  submit() {
    if (this.form.invalid) return;
    this.loading = true;
    const val = this.form.value;

    this.intService.networkIntelligenceSourcesPost({
      name: val.name!,
      type: val.type!,
      url: val.url!,
      enabled: true,
      autoUpdate: false,
      cron: '',
      status: 'Ready',
      lastUpdatedAt: '0001-01-01T00:00:00Z',
      id: '',
      errorMessage: ''
    }).subscribe({
      next: () => {
        this.snackBar.open('配置成功', '关闭', { duration: 3000 });
        this.dialogRef.close(true);
      },
      error: (err) => {
        this.loading = false;
        this.snackBar.open(`保存失败: ${err.error?.message || err.message}`, '关闭', { duration: 3000 });
      }
    });
  }
}
