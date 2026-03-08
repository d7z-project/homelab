import { Component, inject, OnInit, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, Validators, FormArray } from '@angular/forms';
import { MatDialogRef, MatDialogModule, MAT_DIALOG_DATA } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import { DiscoverySelectComponent } from '../../shared/discovery-select.component';
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
    MatIconModule,
    MatTooltipModule,
    DiscoverySelectComponent,
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
          <mat-hint>支持文本 (IP/CIDR) 或 GeoIP (.mmdb, .dat) 文件</mat-hint>
        </mat-form-field>

        <div class="flex gap-4 mt-2">
          <mat-form-field appearance="outline" class="flex-1">
            <mat-label>数据格式</mat-label>
            <mat-select formControlName="format" required>
              <mat-option value="text">文本 (Text/CIDR)</mat-option>
              <mat-option value="csv">CSV (Comma Separated)</mat-option>
              <mat-option value="geoip">GeoIP (MMDB)</mat-option>
              <mat-option value="geoip-dat">GeoIP (V2Ray Dat)</mat-option>
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
        <div
          class="bg-surface-container-low p-4 rounded-2xl border border-outline-variant space-y-4 animate-in fade-in slide-in-from-top-2"
        >
          <div class="text-[10px] font-bold uppercase tracking-wider text-outline mb-2">
            同步配置
          </div>

          @if (form.get('format')?.value === 'text' || form.get('format')?.value === 'csv') {
            <mat-form-field appearance="outline" class="w-full">
              <mat-label>附加标签 (Tags)</mat-label>
              <input matInput formControlName="tags" placeholder="sync, cloud, office" />
              <mat-hint>使用逗号分隔多个标签</mat-hint>
            </mat-form-field>
          }

          @if (form.get('format')?.value === 'csv') {
            <div class="grid grid-cols-3 gap-4">
              <mat-form-field appearance="outline">
                <mat-label>分隔符</mat-label>
                <input matInput formControlName="separator" maxlength="1" />
                <mat-hint>默认为 ,</mat-hint>
              </mat-form-field>
              <mat-form-field appearance="outline">
                <mat-label>IP 列索引</mat-label>
                <input matInput type="number" formControlName="ipColumn" min="0" />
                <mat-hint>从 0 开始</mat-hint>
              </mat-form-field>
              <mat-form-field appearance="outline">
                <mat-label>标签列索引</mat-label>
                <input matInput type="number" formControlName="tagColumn" min="0" />
                <mat-hint>可选</mat-hint>
              </mat-form-field>
            </div>
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

          @if (form.get('format')?.value === 'geoip-dat') {
            <div class="space-y-4">
              <div class="flex items-center justify-between">
                <span class="text-sm">导入全部分类</span>
                <mat-slide-toggle formControlName="importAll"></mat-slide-toggle>
              </div>

              @if (!form.get('importAll')?.value) {
                <mat-form-field appearance="outline" class="w-full">
                  <mat-label>国家/分类代码</mat-label>
                  <input matInput formControlName="code" placeholder="CN" />
                  <mat-hint>例如: CN, US, private, ads</mat-hint>
                </mat-form-field>
              }
            </div>
          }

          <!-- Tag Mapping -->
          <div class="pt-2 border-t border-outline-variant/50">
            <div class="flex items-center justify-between mb-2">
              <div class="text-[10px] font-bold uppercase tracking-wider text-outline">
                标签映射 (Tag Mapping)
              </div>
              <button mat-icon-button (click)="addMapping()" type="button" matTooltip="添加映射">
                <mat-icon class="w-4! h-4! text-sm!">add</mat-icon>
              </button>
            </div>

            <div formArrayName="tagMappings" class="space-y-2">
              @for (m of tagMappings.controls; track $index; let i = $index) {
                <div [formGroupName]="i" class="flex gap-2 items-center">
                  <mat-form-field
                    appearance="outline"
                    class="flex-1 pb-0!"
                    subscriptSizing="dynamic"
                  >
                    <mat-label>原始值</mat-label>
                    <input matInput formControlName="source" placeholder="CN" />
                  </mat-form-field>
                  <mat-icon class="text-outline opacity-40">arrow_forward</mat-icon>
                  <mat-form-field
                    appearance="outline"
                    class="flex-1 pb-0!"
                    subscriptSizing="dynamic"
                  >
                    <mat-label>目标标签</mat-label>
                    <input matInput formControlName="target" placeholder="China" />
                  </mat-form-field>
                  <button mat-icon-button color="warn" (click)="removeMapping(i)" type="button">
                    <mat-icon class="w-4! h-4! text-sm!">remove_circle_outline</mat-icon>
                  </button>
                </div>
              }
              @if (tagMappings.length === 0) {
                <div class="text-center py-2 text-xs text-outline opacity-60 italic">
                  未配置映射，将使用原始值作为标签
                </div>
              }
            </div>
          </div>
        </div>

        <app-discovery-select
          code="network/ip/pools"
          label="目标地址池"
          placeholder="搜索地址池..."
          formControlName="targetGroupId"
          required
        ></app-discovery-select>

        <mat-form-field appearance="outline">
          <mat-label>Cron 表达式</mat-label>
          <input matInput formControlName="cron" required placeholder="0 0 * * *" />
          <mat-hint>标准 5 位 Cron 表达式 (e.g. 0 0 * * *)</mat-hint>
        </mat-form-field>

        <div class="py-2 flex gap-6">
          <mat-slide-toggle formControlName="enabled">启用此策略</mat-slide-toggle>
          <mat-slide-toggle formControlName="allowPrivate">允许私有 IP</mat-slide-toggle>
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
        {{ data.policy ? '保存' : '创建' }}
      </button>
    </mat-dialog-actions>
  `,
  styles: [
    `
      mat-form-field {
        font-size: 13px;
      }
    `,
  ],
})
export class CreateSyncPolicyDialogComponent implements OnInit {
  private fb = inject(FormBuilder);
  private ipService = inject(NetworkIpService);
  private dialogRef = inject(MatDialogRef<CreateSyncPolicyDialogComponent>);
  private snackBar = inject(MatSnackBar);
  public data = inject(MAT_DIALOG_DATA) as { policy?: ModelsIPSyncPolicy };

  loading = false;

  form = this.fb.group({
    name: [this.data.policy?.name || '', Validators.required],
    description: [this.data.policy?.description || ''],
    sourceUrl: [this.data.policy?.sourceUrl || '', Validators.required],
    format: [this.data.policy?.format || 'text', Validators.required],
    mode: [this.data.policy?.mode || 'overwrite', Validators.required],
    targetGroupId: [this.data.policy?.targetGroupId || '', Validators.required],
    cron: [this.data.policy?.cron || '0 0 * * *', Validators.required],
    enabled: [this.data.policy?.enabled ?? true],
    allowPrivate: [this.data.policy?.config?.['allowPrivate'] === 'true'],
    // Specific configs
    tags: [this.data.policy?.config?.['tags'] || this.data.policy?.config?.['tag'] || 'sync'],
    language: [this.data.policy?.config?.['language'] || 'zh-CN'],
    code: [
      this.data.policy?.config?.['code'] === '*' ? '' : this.data.policy?.config?.['code'] || 'CN',
    ],
    importAll: [
      this.data.policy?.config?.['code'] === '*' || this.data.policy?.config?.['code'] === 'all',
    ],
    tagMappings: this.fb.array([]),
    separator: [this.data.policy?.config?.['separator'] || ','],
    ipColumn: [this.data.policy?.config?.['ipColumn'] || 0],
    tagColumn: [this.data.policy?.config?.['tagColumn'] || ''],
  });

  get tagMappings() {
    return this.form.get('tagMappings') as FormArray;
  }

  ngOnInit() {
    this.initMappings();
  }

  initMappings() {
    const mappingStr = this.data.policy?.config?.['tagMapping'];
    if (mappingStr) {
      try {
        const mapping = JSON.parse(mappingStr);
        Object.entries(mapping).forEach(([source, target]) => {
          this.tagMappings.push(
            this.fb.group({
              source: [source, Validators.required],
              target: [target, Validators.required],
            }),
          );
        });
      } catch (e) {
        console.error('Failed to parse tag mapping', e);
      }
    }
  }

  addMapping() {
    this.tagMappings.push(
      this.fb.group({
        source: ['', Validators.required],
        target: ['', Validators.required],
      }),
    );
  }

  removeMapping(index: number) {
    this.tagMappings.removeAt(index);
  }

  submit() {
    if (this.form.invalid) return;
    this.loading = true;
    const val = this.form.value;

    const config: Record<string, string> = {};
    if (val.format === 'text') {
      config['tags'] = val.tags || 'sync';
    } else if (val.format === 'csv') {
      config['tags'] = val.tags || '';
      config['separator'] = val.separator || ',';
      config['ipColumn'] = (val.ipColumn ?? 0).toString();
      if (val.tagColumn !== null && val.tagColumn !== undefined && val.tagColumn !== '') {
        config['tagColumn'] = val.tagColumn.toString();
      }
    } else if (val.format === 'geoip') {
      config['language'] = val.language || 'zh-CN';
    } else if (val.format === 'geoip-dat') {
      config['code'] = val.importAll ? '*' : val.code || 'CN';
    }

    if (val.allowPrivate) {
      config['allowPrivate'] = 'true';
    }

    // Process Tag Mapping
    const mapping: Record<string, string> = {};
    ((val.tagMappings as any[]) || []).forEach((m: any) => {
      if (m.source && m.target) {
        mapping[m.source.trim().toUpperCase()] = m.target.trim();
      }
    });
    if (Object.keys(mapping).length > 0) {
      config['tagMapping'] = JSON.stringify(mapping);
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
        this.snackBar.open(`操作失败: ${err.error?.message || err.message}`, '关闭', {
          duration: 3000,
        });
      },
    });
  }
}
