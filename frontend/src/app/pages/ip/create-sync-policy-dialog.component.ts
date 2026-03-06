import { Component, inject, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, Validators } from '@angular/forms';
import { MatDialogRef, MatDialogModule, MAT_DIALOG_DATA } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { NetworkIpService, ModelsIPSyncPolicy, ModelsIPGroup } from '../../generated';

@Component({
  selector: 'app-create-sync-policy-dialog',
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
    <h2 mat-dialog-title>{{ data.policy ? '编辑同步策略' : '新建同步策略' }}</h2>
    <mat-dialog-content>
      <form [formGroup]="form" class="flex flex-col gap-4 pt-2">
        <mat-form-field appearance="outline">
          <mat-label>策略名称</mat-label>
          <input matInput formControlName="name" required />
        </mat-form-field>

        <mat-form-field appearance="outline">
          <mat-label>描述</mat-label>
          <textarea matInput formControlName="description" rows="2"></textarea>
        </mat-form-field>

        <mat-form-field appearance="outline">
          <mat-label>源 URL</mat-label>
          <input matInput formControlName="sourceUrl" required placeholder="https://..." />
          <mat-hint>支持文本 (IP/CIDR) 或 GeoIP (.mmdb) 文件</mat-hint>
        </mat-form-field>

        <div class="flex gap-4 mt-2">
          <mat-form-field appearance="outline" class="flex-1">
            <mat-label>数据格式</mat-label>
            <mat-select formControlName="format" required>
              <mat-option value="text">文本 (Text/CIDR)</mat-option>
              <mat-option value="geoip">GeoIP (MMDB)</mat-option>
            </mat-select>
          </mat-form-field>

          <mat-form-field appearance="outline" class="flex-1">
            <mat-label>导入模式</mat-label>
            <mat-select formControlName="mode" required>
              <mat-option value="overwrite">覆盖 (Overwrite)</mat-option>
              <mat-option value="append">追加 (Append)</mat-option>
            </mat-select>
          </mat-form-field>
        </div>

        <!-- Format Specific Config -->
        <div class="bg-surface-container-low p-4 rounded-2xl border border-outline-variant space-y-2 animate-in fade-in slide-in-from-top-2">
          <div class="text-[10px] font-bold uppercase tracking-wider text-outline mb-2">配置参数</div>
          
          @if (form.get('format')?.value === 'text') {
            <mat-form-field appearance="outline" class="w-full">
              <mat-label>默认标签</mat-label>
              <input matInput formControlName="tag" placeholder="sync" />
              <mat-hint>同步进来的条目将带上此标签</mat-hint>
            </mat-form-field>
          }

          @if (form.get('format')?.value === 'geoip') {
            <mat-form-field appearance="outline" class="w-full">
              <mat-label>语言偏好</mat-label>
              <mat-select formControlName="language">
                <mat-option value="zh-CN">简体中文 (Simplified Chinese)</mat-option>
                <mat-option value="en">英语 (English)</mat-option>
                <mat-option value="ja">日语 (Japanese)</mat-option>
                <mat-option value="de">德语 (German)</mat-option>
                <mat-option value="fr">法语 (French)</mat-option>
              </mat-select>
              <mat-hint>MMDB 记录中提取城市名称时的首选语言</mat-hint>
            </mat-form-field>
          }
        </div>

        <mat-form-field appearance="outline">
          <mat-label>目标地址池</mat-label>
          <mat-select formControlName="targetGroupId" required>
            <mat-option *ngFor="let p of pools()" [value]="p.id">{{ p.name }}</mat-option>
          </mat-select>
        </mat-form-field>

        <mat-form-field appearance="outline">
          <mat-label>Cron 表达式</mat-label>
          <input matInput formControlName="cron" required placeholder="0 0 * * *" />
          <mat-hint>标准 5 位 Cron 表达式 (e.g. 0 0 * * *)</mat-hint>
        </mat-form-field>

        <div class="py-2">
          <mat-slide-toggle formControlName="enabled">启用此策略</mat-slide-toggle>
        </div>
      </form>
    </mat-dialog-content>
    <mat-dialog-actions align="end">
      <button mat-button mat-dialog-close>取消</button>
      <button mat-flat-button color="primary" [disabled]="form.invalid || loading" (click)="submit()">
        {{ data.policy ? '保存' : '创建' }}
      </button>
    </mat-dialog-actions>
  `,
})
export class CreateSyncPolicyDialogComponent implements OnInit {
  private fb = inject(FormBuilder);
  private ipService = inject(NetworkIpService);
  private dialogRef = inject(MatDialogRef<CreateSyncPolicyDialogComponent>);
  private snackBar = inject(MatSnackBar);
  public data = inject(MAT_DIALOG_DATA) as { policy?: ModelsIPSyncPolicy };

  loading = false;
  pools = signal<ModelsIPGroup[]>([]);

  form = this.fb.group({
    name: [this.data.policy?.name || '', Validators.required],
    description: [this.data.policy?.description || ''],
    sourceUrl: [this.data.policy?.sourceUrl || '', Validators.required],
    format: [this.data.policy?.format || 'text', Validators.required],
    mode: [this.data.policy?.mode || 'overwrite', Validators.required],
    targetGroupId: [this.data.policy?.targetGroupId || '', Validators.required],
    cron: [this.data.policy?.cron || '0 0 * * *', Validators.required],
    enabled: [this.data.policy?.enabled ?? true],
    // Specific configs
    tag: [this.data.policy?.config?.['tag'] || 'sync'],
    language: [this.data.policy?.config?.['language'] || 'zh-CN'],
  });

  ngOnInit() {
    this.loadPools();
  }

  loadPools() {
    this.ipService.networkIpPoolsGet(1, 1000).subscribe({
      next: (res) => this.pools.set(res.items || []),
    });
  }

  submit() {
    if (this.form.invalid) return;
    this.loading = true;
    const val = this.form.value;

    const config: Record<string, string> = {};
    if (val.format === 'text') {
      config['tag'] = val.tag || 'sync';
    } else if (val.format === 'geoip') {
      config['language'] = val.language || 'zh-CN';
    }

    const policy: ModelsIPSyncPolicy = {
      ...this.data.policy,
      name: val.name!,
      description: val.description || '',
      sourceUrl: val.sourceUrl!,
      format: val.format!,
      mode: val.mode!,
      config: config,
      targetGroupId: val.targetGroupId!,
      cron: val.cron!,
      enabled: !!val.enabled,
    };

    const obs = this.data.policy?.id
      ? this.ipService.networkIpSyncIdPut(this.data.policy.id, policy)
      : this.ipService.networkIpSyncPost(policy);

    obs.subscribe({
      next: () => {
        this.snackBar.open(this.data.policy ? '更新成功' : '创建成功', '关闭', { duration: 3000 });
        this.dialogRef.close(true);
      },
      error: (err) => {
        this.loading = false;
        this.snackBar.open(`操作失败: ${err.error?.message || err.message}`, '关闭', { duration: 3000 });
      },
    });
  }
}
