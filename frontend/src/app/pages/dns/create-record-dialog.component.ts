import { Component, Inject, inject, ChangeDetectorRef, AfterViewInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { MatIconModule } from '@angular/material/icon';
import { FormsModule } from '@angular/forms';
import { ModelsDomain, ModelsRecord } from '../../generated';

import { DiscoverySelectComponent } from '../../shared/discovery-select.component';

@Component({
  selector: 'app-create-record-dialog',
  standalone: true,
  imports: [
    CommonModule,
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatSlideToggleModule,
    MatIconModule,
    FormsModule,
    DiscoverySelectComponent,
  ],
  template: `
    <h2 mat-dialog-title class="pt-6!">
      <mat-icon class="mr-2 align-middle text-primary">layers</mat-icon>
      {{ isEdit ? '修改解析记录' : '新增解析记录' }}
    </h2>
    <mat-dialog-content style="min-width: 350px; max-width: 600px;">
      <div class="pt-3 space-y-4">
        <!-- Domain Discovery Select -->
        <app-discovery-select
          code="network/dns/domains"
          label="所属域名"
          placeholder="搜索域名..."
          [(ngModel)]="record.domainId"
          [disabled]="isEdit"
          required
        ></app-discovery-select>

        <div class="flex gap-4">
          <mat-form-field appearance="outline" class="flex-1">
            <mat-label>主机记录 (Name)</mat-label>
            <input
              matInput
              [(ngModel)]="record.name"
              placeholder="例如: www 或 @"
              required
              [disabled]="isEdit && record.type === 'SOA'"
              pattern="^(@|[\\-a-zA-Z0-9\\*_]+(\\.[\\-a-zA-Z0-9\\*_]+)*)$"
              #nameInput="ngModel"
            />
            <mat-hint>&#64; 表示主域名</mat-hint>
            @if (nameInput.errors?.['required']) {
              <mat-error>请输入主机记录</mat-error>
            }
            @if (nameInput.errors?.['pattern']) {
              <mat-error>主机记录格式不正确</mat-error>
            }
          </mat-form-field>

          <mat-form-field appearance="outline" class="w-32">
            <mat-label>记录类型</mat-label>
            <mat-select
              [(ngModel)]="record.type"
              (selectionChange)="onTypeChange()"
              [disabled]="isEdit && record.type === 'SOA'"
            >
              @for (t of recordTypes; track t) {
                <mat-option [value]="t">{{ t }}</mat-option>
              }
            </mat-select>
          </mat-form-field>
        </div>

        @if (!['SOA', 'SRV', 'CAA'].includes(record.type || '')) {
          <mat-form-field appearance="outline" class="w-full">
            <mat-label>记录值 (Value)</mat-label>
            <input
              matInput
              [(ngModel)]="record.value"
              [placeholder]="getValuePlaceholder()"
              required
              #valueInput="ngModel"
            />
            @if (valueInput.errors?.['required']) {
              <mat-error>请输入记录值</mat-error>
            }
            @if (record.type === 'A' && !isValidIPv4()) {
              <mat-error>无效的 IPv4 地址</mat-error>
            }
            @if (record.type === 'AAAA' && !isValidIPv6()) {
              <mat-error>无效的 IPv6 地址</mat-error>
            }
          </mat-form-field>
        }

        <!-- SOA specific fields -->
        @if (record.type === 'SOA') {
          <div class="space-y-4 animate-in fade-in slide-in-from-top-2 duration-300">
            <div class="flex gap-4">
              <mat-form-field appearance="outline" class="flex-1">
                <mat-label>主名称服务器 (MNAME)</mat-label>
                <input
                  matInput
                  [(ngModel)]="soaMname"
                  (ngModelChange)="syncValue()"
                  required
                  placeholder="例如: ns1.hover.com."
                  #soaMnameInput="ngModel"
                  pattern="^([a-z0-9]+(-[a-z0-9]+)*\\.)+[a-z]{2,}\\.?$"
                />
                @if (soaMnameInput.errors?.['pattern']) {
                  <mat-error>格式不正确</mat-error>
                }
              </mat-form-field>
              <mat-form-field appearance="outline" class="flex-1">
                <mat-label>负责人邮箱 (RNAME)</mat-label>
                <input
                  matInput
                  [(ngModel)]="soaRname"
                  (ngModelChange)="syncValue()"
                  required
                  placeholder="例如: admin.example.com."
                  #soaRnameInput="ngModel"
                  pattern="^([a-z0-9]+(-[a-z0-9]+)*\\.)+[a-z]{2,}\\.?$"
                />
                <mat-hint>邮箱中的 &#64; 需替换为 .</mat-hint>
                @if (soaRnameInput.errors?.['pattern']) {
                  <mat-error>格式不正确</mat-error>
                }
              </mat-form-field>
            </div>

            <div class="grid grid-cols-2 sm:grid-cols-3 gap-4">
              <mat-form-field appearance="outline">
                <mat-label>序列号 (SERIAL)</mat-label>
                <input matInput [value]="soaSerial" disabled />
                <mat-hint>系统自动更新</mat-hint>
              </mat-form-field>
              <mat-form-field appearance="outline">
                <mat-label>刷新间隔 (REFRESH)</mat-label>
                <input matInput [value]="soaRefresh" disabled />
              </mat-form-field>
              <mat-form-field appearance="outline">
                <mat-label>重试间隔 (RETRY)</mat-label>
                <input matInput [value]="soaRetry" disabled />
              </mat-form-field>
              <mat-form-field appearance="outline">
                <mat-label>过期时间 (EXPIRE)</mat-label>
                <input matInput [value]="soaExpire" disabled />
              </mat-form-field>
              <mat-form-field appearance="outline">
                <mat-label>最小 TTL (MINIMUM)</mat-label>
                <input matInput [value]="soaMinimum" disabled />
              </mat-form-field>
            </div>

            <div class="p-3 bg-info-container text-on-info-container rounded-xl text-xs flex gap-2">
              <mat-icon class="text-sm h-4 w-4">info</mat-icon>
              <span>提示: SOA 记录仅支持修改 MNAME 和 RNAME，其余字段由系统维护。</span>
            </div>
          </div>
        }

        <!-- SRV specific fields -->
        @if (record.type === 'SRV') {
          <div class="space-y-4 animate-in fade-in slide-in-from-top-2 duration-300">
            <div class="grid grid-cols-2 gap-4">
              <mat-form-field appearance="outline">
                <mat-label>权重 (Weight)</mat-label>
                <input
                  matInput
                  type="number"
                  [(ngModel)]="srvWeight"
                  (ngModelChange)="syncValue()"
                  min="0"
                  max="65535"
                  required
                />
              </mat-form-field>
              <mat-form-field appearance="outline">
                <mat-label>端口 (Port)</mat-label>
                <input
                  matInput
                  type="number"
                  [(ngModel)]="srvPort"
                  (ngModelChange)="syncValue()"
                  min="0"
                  max="65535"
                  required
                />
              </mat-form-field>
            </div>
            <mat-form-field appearance="outline" class="w-full">
              <mat-label>目标主机 (Target)</mat-label>
              <input
                matInput
                [(ngModel)]="srvTarget"
                (ngModelChange)="syncValue()"
                required
                placeholder="例如: server.example.com."
              />
            </mat-form-field>
          </div>
        }

        <!-- CAA specific fields -->
        @if (record.type === 'CAA') {
          <div class="space-y-4 animate-in fade-in slide-in-from-top-2 duration-300">
            <div class="grid grid-cols-2 gap-4">
              <mat-form-field appearance="outline">
                <mat-label>标志 (Flags)</mat-label>
                <input
                  matInput
                  type="number"
                  [(ngModel)]="caaFlags"
                  (ngModelChange)="syncValue()"
                  min="0"
                  max="255"
                  required
                />
                <mat-hint>通常为 0</mat-hint>
              </mat-form-field>
              <mat-form-field appearance="outline">
                <mat-label>标签 (Tag)</mat-label>
                <mat-select [(ngModel)]="caaTag" (selectionChange)="syncValue()">
                  <mat-option value="issue">issue (允许特定 CA 颁发证书)</mat-option>
                  <mat-option value="issuewild">issuewild (允许颁发泛域名证书)</mat-option>
                  <mat-option value="iodef">iodef (CA 报告违规通知 URL)</mat-option>
                </mat-select>
              </mat-form-field>
            </div>
            <mat-form-field appearance="outline" class="w-full">
              <mat-label>CA 域名或 URL</mat-label>
              <input
                matInput
                [(ngModel)]="caaValue"
                (ngModelChange)="syncValue()"
                required
                placeholder="例如: letsencrypt.org"
              />
            </mat-form-field>
          </div>
        }

        <div class="flex gap-4">
          <mat-form-field appearance="outline" class="flex-1">
            <mat-label>TTL (秒)</mat-label>
            <input
              matInput
              type="number"
              [(ngModel)]="record.ttl"
              min="1"
              required
              placeholder="默认 600"
              #ttlInput="ngModel"
            />
            <mat-hint>推荐值: 600, 3600, 86400</mat-hint>
            @if (ttlInput.errors?.['required'] || (record.ttl !== undefined && record.ttl < 1)) {
              <mat-error>TTL 必须大于 0</mat-error>
            }
          </mat-form-field>

          @if (record.type === 'MX' || record.type === 'SRV') {
            <mat-form-field appearance="outline" class="w-32">
              <mat-label>优先级</mat-label>
              <input
                matInput
                type="number"
                [(ngModel)]="record.priority"
                (ngModelChange)="syncValue()"
                min="0"
                max="65535"
              />
            </mat-form-field>
          }
        </div>

        <div
          class="flex items-center justify-between p-4 bg-surface-container-low rounded-2xl border border-outline-variant/30"
        >
          <div class="flex flex-col">
            <span class="text-sm font-bold">启用状态</span>
            <span class="text-xs text-outline">禁用后此条记录将不再参与解析</span>
          </div>
          <mat-slide-toggle
            color="primary"
            [(ngModel)]="record.enabled"
            [disabled]="record.type === 'SOA'"
          >
          </mat-slide-toggle>
        </div>

        @if (record.type === 'CNAME') {
          <div class="p-3 bg-warn-container text-on-warn-container rounded-xl text-xs flex gap-2">
            <mat-icon class="text-sm h-4 w-4">info</mat-icon>
            <span>提示: CNAME 记录不能与同一主机记录下的其他记录（如 A, TXT）共存。</span>
          </div>
        }
      </div>
    </mat-dialog-content>
    <mat-dialog-actions align="end" class="px-6! pb-6!">
      <button mat-button mat-dialog-close>取消</button>
      <button
        mat-flat-button
        color="primary"
        (click)="confirm()"
        [disabled]="!isValid()"
        class="ml-2! px-6 rounded-full"
      >
        <mat-icon class="mr-1">check</mat-icon>
        {{ isEdit ? '保存更改' : '确定添加' }}
      </button>
    </mat-dialog-actions>
  `,
})
export class CreateRecordDialogComponent implements AfterViewInit {
  private cdr = inject(ChangeDetectorRef);
  private dialogRef = inject(MatDialogRef<CreateRecordDialogComponent>);
  isEdit = false;
  record: ModelsRecord = {
    domainId: '',
    name: '',
    type: 'A',
    value: '',
    ttl: 600,
    priority: 10,
    enabled: true,
  };
  recordTypes = ['A', 'AAAA', 'CNAME', 'MX', 'TXT', 'NS', 'SRV', 'CAA'];

  // SOA parts
  soaMname = '';
  soaRname = '';
  soaSerial = '';
  soaRefresh = '';
  soaRetry = '';
  soaExpire = '';
  soaMinimum = '';

  // SRV parts (Priority is handled by record.priority)
  srvWeight = 0;
  srvPort = 0;
  srvTarget = '';

  // CAA parts
  caaFlags = 0;
  caaTag = 'issue';
  caaValue = '';

  constructor(
    @Inject(MAT_DIALOG_DATA)
    public data: { record: ModelsRecord | null; defaultDomainId?: string },
  ) {
    if (data.record) {
      this.isEdit = true;
      this.record = { ...data.record };
      if (this.record.type === 'SOA') {
        if (!this.recordTypes.includes('SOA')) {
          this.recordTypes.push('SOA');
        }
      }
      this.parseValue();
    } else if (data.defaultDomainId) {
      this.record.domainId = data.defaultDomainId;
    }
  }

  ngAfterViewInit() {
    // Ensure view state is stable after potential initial value syncing and child component init
    setTimeout(() => {
      this.cdr.detectChanges();
    });
  }

  onTypeChange() {
    this.record.value = '';
    this.syncValue();
  }

  parseValue() {
    if (!this.record.value) return;
    const parts = this.record.value.split(/\s+/);

    if (this.record.type === 'SOA') {
      if (parts.length >= 7) {
        this.soaMname = parts[0];
        this.soaRname = parts[1];
        this.soaSerial = parts[2];
        this.soaRefresh = parts[3];
        this.soaRetry = parts[4];
        this.soaExpire = parts[5];
        this.soaMinimum = parts[6];
      }
    } else if (this.record.type === 'SRV') {
      if (parts.length >= 3) {
        this.srvWeight = parseInt(parts[0]) || 0;
        this.srvPort = parseInt(parts[1]) || 0;
        this.srvTarget = parts.slice(2).join(' ');
      }
    } else if (this.record.type === 'CAA') {
      if (parts.length >= 3) {
        this.caaFlags = parseInt(parts[0]) || 0;
        this.caaTag = parts[1] || 'issue';
        // Handle potentially quoted values
        let val = parts.slice(2).join(' ');
        if (val.startsWith('"') && val.endsWith('"')) {
          val = val.substring(1, val.length - 1);
        }
        this.caaValue = val;
      }
    }
  }

  ensureTrailingDot(val: string): string {
    if (!val) return '';
    val = val.trim();
    return val.endsWith('.') ? val : val + '.';
  }

  syncValue() {
    if (this.record.type === 'SOA') {
      const mname = this.ensureTrailingDot(this.soaMname);
      const rname = this.ensureTrailingDot(this.soaRname);
      this.record.value =
        `${mname} ${rname} ${this.soaSerial} ${this.soaRefresh} ${this.soaRetry} ${this.soaExpire} ${this.soaMinimum}`.trim();
    } else if (this.record.type === 'SRV') {
      const target = this.ensureTrailingDot(this.srvTarget);
      this.record.value = `${this.srvWeight} ${this.srvPort} ${target}`.trim();
    } else if (this.record.type === 'CAA') {
      this.record.value = `${this.caaFlags} ${this.caaTag} "${this.caaValue}"`;
    } else if (
      this.record.type === 'CNAME' ||
      this.record.type === 'MX' ||
      this.record.type === 'NS'
    ) {
      this.record.value = this.ensureTrailingDot(this.record.value || '');
    }
  }

  getValuePlaceholder(): string {
    switch (this.record.type) {
      case 'A':
        return 'IPv4 地址, 如 1.2.3.4';
      case 'AAAA':
        return 'IPv6 地址, 如 2001:db8::1';
      case 'CNAME':
        return '别名域名, 如 example.com.';
      case 'MX':
        return '邮件服务器, 如 mail.example.com.';
      case 'SOA':
        return 'ns1.example.com. admin.example.com. 2026030301 7200 3600 1209600 3600';
      default:
        return '记录内容...';
    }
  }

  isValidIPv4(): boolean {
    const ipv4Regex =
      /^(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/;
    return ipv4Regex.test(this.record.value || '');
  }

  isValidIPv6(): boolean {
    const ipv6Regex =
      /^([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}$|^(([0-9a-fA-F]{1,4}:){0,6}[0-9a-fA-F]{1,4})?::(([0-9a-fA-F]{1,4}:){0,6}[0-9a-fA-F]{1,4})?$/;
    return ipv6Regex.test(this.record.value || '');
  }

  isValid(): boolean {
    const name = this.record.name?.trim();
    if (!name || this.record.domainId! || !this.record.type) return false;
    if (this.record.ttl === undefined || this.record.ttl < 1) return false;

    // Value must exist (either directly or via parts)
    if (
      this.record.value! &&
      this.record.type !== 'SOA' &&
      this.record.type !== 'SRV' &&
      this.record.type !== 'CAA'
    )
      return false;

    // Type-specific value validation
    if (this.record.type === 'A' && !this.isValidIPv4()) return false;
    if (this.record.type === 'AAAA' && !this.isValidIPv6()) return false;

    if (this.record.type === 'SOA') {
      const dnsPattern = /^([a-z0-9]+(-[a-z0-9]+)*\.)+[a-z]{2,}\.?$/i;
      if (!dnsPattern.test(this.soaMname) || !dnsPattern.test(this.soaRname)) return false;
    }

    if (this.record.type === 'SRV') {
      if (this.srvWeight < 0 || this.srvWeight > 65535) return false;
      if (this.srvPort < 0 || this.srvPort > 65535) return false;
      if (!this.srvTarget) return false;
      if (
        this.record.priority === undefined ||
        this.record.priority < 0 ||
        this.record.priority > 65535
      )
        return false;
    }

    if (this.record.type === 'CAA') {
      if (this.caaFlags < 0 || this.caaFlags > 255) return false;
      if (!this.caaValue) return false;
    }

    // Basic name pattern check
    const namePattern = /^(@|[\\-a-zA-Z0-9\\*_]+(\\.[\\-a-zA-Z0-9\\*_]+)*)$/;
    return namePattern.test(name);
  }

  confirm() {
    if (this.isValid()) {
      this.dialogRef.close(this.record);
    }
  }
}
