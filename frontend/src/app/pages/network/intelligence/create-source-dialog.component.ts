import { Component, inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, Validators } from '@angular/forms';
import { MatDialogRef, MatDialogModule, MAT_DIALOG_DATA } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { MatSnackBar } from '@angular/material/snack-bar';
import { NetworkIntelligenceService, ModelsIntelligenceSource } from '../../../generated';

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
    MatSlideToggleModule,
  ],
  template: `
    <h2 mat-dialog-title>{{ data ? '编辑' : '配置' }}情报数据源</h2>
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

        <div
          class="flex flex-col gap-2 p-4 bg-surface-container-low rounded-2xl border border-outline-variant"
        >
          <div class="flex gap-6">
            <mat-slide-toggle formControlName="autoUpdate">启用自动更新</mat-slide-toggle>
            <mat-slide-toggle formControlName="allowPrivate">允许私有 IP</mat-slide-toggle>
          </div>

          @if (form.get('autoUpdate')?.value) {
            <mat-form-field appearance="outline" class="mt-2">
              <mat-label>Cron 表达式</mat-label>
              <input matInput formControlName="cron" placeholder="0 0 * * *" required />
              <mat-hint>标准 Cron 格式 (分 时 日 月 周)</mat-hint>
            </mat-form-field>
          }
        </div>
      </form>
    </mat-dialog-content>
    <mat-dialog-actions align="end">
      <button mat-button mat-dialog-close>取消</button>
      <button
        mat-flat-button
        color="primary"
        [disabled]="form.invalid || loading"
        (click)="submit()"
      >
        保存配置
      </button>
    </mat-dialog-actions>
  `,
})
export class CreateSourceDialogComponent implements OnInit {
  private fb = inject(FormBuilder);
  private intService = inject(NetworkIntelligenceService);
  private dialogRef = inject(MatDialogRef<CreateSourceDialogComponent>);
  public data = inject<ModelsIntelligenceSource | null>(MAT_DIALOG_DATA, { optional: true });
  private snackBar = inject(MatSnackBar);

  loading = false;

  form = this.fb.group({
    name: ['', Validators.required],
    type: ['asn', Validators.required],
    url: ['', [Validators.required, Validators.pattern(/^https?:\/\/.+$/)]],
    autoUpdate: [false],
    cron: [''],
    allowPrivate: [false],
  });

  ngOnInit() {
    if (this.data) {
      this.form.patchValue({
        name: this.data.name,
        type: this.data.type,
        url: this.data.url,
        autoUpdate: this.data.autoUpdate,
        cron: this.data.cron,
        allowPrivate: this.data.config?.['allowPrivate'] === 'true',
      });
      if (this.data.autoUpdate) {
        this.form.get('cron')?.setValidators(Validators.required);
      }
    }

    // Dynamic validation for cron
    this.form.get('autoUpdate')?.valueChanges.subscribe((val) => {
      const cronControl = this.form.get('cron');
      if (val) {
        cronControl?.setValidators(Validators.required);
        if (!cronControl?.value) cronControl?.setValue('0 0 * * *');
      } else {
        cronControl?.clearValidators();
      }
      cronControl?.updateValueAndValidity();
    });
  }

  submit() {
    if (this.form.invalid) return;
    this.loading = true;
    const val = this.form.value;

    const config: Record<string, string> = { ...this.data?.config };
    if (val.allowPrivate) {
      config['allowPrivate'] = 'true';
    } else {
      delete config['allowPrivate'];
    }

    const payload: ModelsIntelligenceSource = {
      name: val.name!,
      type: val.type!,
      url: val.url!,
      enabled: this.data ? this.data.enabled : true,
      autoUpdate: !!val.autoUpdate,
      cron: val.cron || '',
      status: this.data ? this.data.status : 'Ready',
      lastUpdatedAt: this.data ? this.data.lastUpdatedAt : '0001-01-01T00:00:00Z',
      id: this.data ? this.data.id : '',
      errorMessage: this.data ? this.data.errorMessage : '',
      config: config,
    };

    const obs =
      this.data && this.data.id
        ? this.intService.networkIntelligenceSourcesIdPut(this.data.id, payload)
        : this.intService.networkIntelligenceSourcesPost(payload);

    obs.subscribe({
      next: () => {
        this.snackBar.open(this.data ? '修改成功' : '配置成功', '关闭', { duration: 3000 });
        this.dialogRef.close(true);
      },
      error: (err) => {
        this.loading = false;
        this.snackBar.open(`保存失败: ${err.error?.message || err.message}`, '关闭', {
          duration: 3000,
        });
      },
    });
  }
}
