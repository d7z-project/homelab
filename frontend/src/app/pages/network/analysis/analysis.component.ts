import { Component, OnInit, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatDividerModule } from '@angular/material/divider';
import { MatTooltipModule } from '@angular/material/tooltip';
import { 
  NetworkIpService, 
  NetworkSiteService, 
  ModelsIPAnalysisResult, 
  ModelsIPInfoResponse, 
  ModelsSiteAnalysisResult 
} from '../../../generated';
import { PageHeaderComponent } from '../../../shared/page-header.component';

@Component({
  selector: 'app-analysis',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    MatCardModule,
    MatButtonModule,
    MatIconModule,
    MatInputModule,
    MatFormFieldModule,
    MatProgressSpinnerModule,
    MatDividerModule,
    MatTooltipModule,
    PageHeaderComponent,
  ],
  templateUrl: './analysis.component.html',
  styles: [`
    .search-field-m3 {
      ::ng-deep .mdc-text-field--filled { background-color: transparent !important; }
      ::ng-deep .mdc-line-ripple { display: none; }
      ::ng-deep .mat-mdc-form-field-subscript-wrapper { display: none; }
      ::ng-deep .mat-mdc-text-field-wrapper { padding-bottom: 0; }
    }
  `]
})
export class AnalysisComponent {
  private ipService = inject(NetworkIpService);
  private siteService = inject(NetworkSiteService);
  private snackBar = inject(MatSnackBar);

  query = signal('');
  loading = signal(false);

  // Results
  ipResult = signal<ModelsIPAnalysisResult | null>(null);
  ipInfo = signal<ModelsIPInfoResponse | null>(null);
  siteResult = signal<ModelsSiteAnalysisResult | null>(null);

  hasResult = computed(() => !!(this.ipResult() || this.ipInfo() || this.siteResult()));

  runAnalysis() {
    const val = this.query().trim();
    if (!val) return;

    this.loading.set(true);
    this.ipResult.set(null);
    this.ipInfo.set(null);
    this.siteResult.set(null);

    const isIP = this.checkIsIP(val);

    if (isIP) {
      this.analyzeIP(val);
    } else {
      this.analyzeDomain(val);
    }
  }

  checkIsIP(val: string): boolean {
    // Basic IP detection (IPv4 or IPv6)
    const ipv4Regex = /^(\d{1,3}\.){3}\d{1,3}$/;
    const ipv6Regex = /^([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}$|^(([0-9a-fA-F]{1,4}:){0,6}[0-9a-fA-F]{1,4})?::(([0-9a-fA-F]{1,4}:){0,6}[0-9a-fA-F]{1,4})?$/;
    return ipv4Regex.test(val) || ipv6Regex.test(val);
  }

  private analyzeIP(ip: string) {
    this.ipService.networkIpAnalysisHitTestPost({ ip, groupIds: [] }).subscribe({
      next: (res) => this.ipResult.set(res),
      error: (err) => this.handleError(err)
    });

    this.ipService.networkIpAnalysisInfoGet(ip).subscribe({
      next: (res) => {
        this.ipInfo.set(res);
        this.loading.set(false);
      },
      error: () => this.loading.set(false)
    });
  }

  private analyzeDomain(domain: string) {
    this.siteService.networkSiteAnalysisHitTestPost({ domain, groupIds: [] }).subscribe({
      next: (res) => {
        this.siteResult.set(res);
        this.loading.set(false);
      },
      error: (err) => {
        this.handleError(err);
        this.loading.set(false);
      }
    });
  }

  private handleError(err: any) {
    this.snackBar.open(`分析失败: ${err.error?.message || err.message}`, '关闭', { duration: 3000 });
  }
}
