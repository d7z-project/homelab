import { Component, OnInit, inject, signal, computed, ViewChild, ElementRef, effect } from '@angular/core';
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
import { HttpClient } from '@angular/common/http';
import { Router, ActivatedRoute } from '@angular/router';
import { 
  NetworkIpService, 
  NetworkSiteService, 
  ModelsIPAnalysisResult, 
  ModelsIPInfoResponse, 
  ModelsSiteAnalysisResult 
} from '../../../generated';
import { PageHeaderComponent } from '../../../shared/page-header.component';
import * as echarts from 'echarts';

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
    .map-wrapper {
      width: 100%;
      aspect-ratio: 16 / 9;
      min-height: 300px;
    }
    .clickable-link {
      cursor: pointer;
      &:hover {
        text-decoration: underline;
        color: var(--mat-sys-primary);
      }
    }
  `]
})
export class AnalysisComponent implements OnInit {
  @ViewChild('mapContainer') set mapContainer(content: ElementRef) {
    if (content) {
      this._mapContainer = content;
      this.refreshMap();
    }
  }
  private _mapContainer?: ElementRef;
  
  private ipService = inject(NetworkIpService);
  private siteService = inject(NetworkSiteService);
  private snackBar = inject(MatSnackBar);
  private http = inject(HttpClient);
  private router = inject(Router);
  private route = inject(ActivatedRoute);

  query = signal('');
  loading = signal(false);
  hitTestLoading = signal(false);

  // Results
  ipResult = signal<ModelsIPAnalysisResult | null>(null);
  ipInfo = signal<ModelsIPInfoResponse | null>(null);
  siteResult = signal<ModelsSiteAnalysisResult | null>(null);
  
  private chart?: echarts.ECharts;

  hasResult = computed(() => !!(this.ipResult() || this.ipInfo() || this.siteResult() || this.hitTestLoading()));

  constructor() {
    effect(() => {
      this.ipInfo();
      this.siteResult();
      this.refreshMap();
    });
  }

  private refreshMap() {
    if (!this._mapContainer) return;
    
    const info = this.ipInfo();
    const site = this.siteResult();
    
    let infos: ModelsIPInfoResponse[] = [];
    if (info) {
      infos = [info];
    } else if (site && site.dns) {
      if (site.dns.a) infos.push(...site.dns.a);
      if (site.dns.aaaa) infos.push(...site.dns.aaaa);
    }

    if (infos.length > 0) {
      setTimeout(() => this.initChart(this._mapContainer!.nativeElement, infos), 200);
    }
  }

  ngOnInit() {
    this.route.queryParams.subscribe(params => {
      const q = params['q'];
      if (q && q !== this.query()) {
        this.query.set(q);
        this.runAnalysis(false);
      }
    });
  }

  private initChart(element: HTMLElement, infos: ModelsIPInfoResponse[]): void {
    echarts.getInstanceByDom(element)?.dispose();
    
    const chart = echarts.init(element);
    this.chart = chart;
    
    this.http.get('maps/world.json').subscribe(geoJson => {
      echarts.registerMap('world', geoJson as any);
      
      const scatterData = infos.map(info => {
        const coords = info.location?.split(',').map(v => parseFloat(v)) || [0, 0];
        return {
          name: info.ip,
          value: [coords[1], coords[0]]
        };
      });

      const center = scatterData.length > 0 ? scatterData[0].value : [0, 0];

      const option: echarts.EChartsOption = {
        backgroundColor: 'transparent',
        geo: {
          map: 'world',
          roam: true,
          label: { show: false },
          emphasis: {
            label: { show: false },
            itemStyle: { areaColor: '#3b82f6' }
          },
          itemStyle: {
            areaColor: '#1e293b',
            borderColor: '#334155',
            borderWidth: 0.5
          },
          zoom: 1.5,
          center: center
        },
        series: [
          {
            type: 'effectScatter',
            coordinateSystem: 'geo',
            data: scatterData,
            symbolSize: 12,
            showEffectOn: 'render',
            rippleEffect: { brushType: 'stroke', scale: 4 },
            label: {
              formatter: '{b}',
              position: 'right',
              show: true,
              color: '#fff',
              fontWeight: 'bold',
              backgroundColor: 'rgba(0,0,0,0.7)',
              padding: [4, 8],
              borderRadius: 4
            },
            itemStyle: { color: '#3b82f6', shadowBlur: 10, shadowColor: '#3b82f6' },
            zlevel: 1
          }
        ]
      };
      chart.setOption(option);
      
      const resizeObserver = new ResizeObserver(() => chart.resize());
      resizeObserver.observe(element);
    });
  }

  runAnalysis(updateUrl = true) {
    const val = this.query().trim();
    if (!val) return;

    if (updateUrl) {
      this.router.navigate([], {
        relativeTo: this.route,
        queryParams: { q: val },
        queryParamsHandling: 'merge'
      });
    }

    this.loading.set(true);
    this.hitTestLoading.set(true);
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

  pivotSearch(val: string | undefined) {
    if (!val) return;
    // Strip trailing dots often found in CNAME/SOA
    const cleaned = val.replace(/\.+$/, '');
    this.query.set(cleaned);
    this.runAnalysis();
  }

  checkIsIP(val: string): boolean {
    const ipv4Regex = /^(\d{1,3}\.){3}\d{1,3}$/;
    const ipv6Regex = /^([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}$|^(([0-9a-fA-F]{1,4}:){0,6}[0-9a-fA-F]{1,4})?::(([0-9a-fA-F]{1,4}:){0,6}[0-9a-fA-F]{1,4})?$/;
    return ipv4Regex.test(val) || ipv6Regex.test(val);
  }

  private analyzeIP(ip: string) {
    this.ipService.networkIpAnalysisHitTestPost({ ip, groupIds: [] }).subscribe({
      next: (res) => {
        this.ipResult.set(res);
        this.hitTestLoading.set(false);
      },
      error: (err) => {
        this.handleError(err);
        this.hitTestLoading.set(false);
      }
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
        this.hitTestLoading.set(false);
        this.loading.set(false);
      },
      error: (err) => {
        this.handleError(err);
        this.hitTestLoading.set(false);
        this.loading.set(false);
      }
    });
  }

  private handleError(err: any) {
    this.snackBar.open(`分析失败: ${err.error?.message || err.message}`, '关闭', { duration: 3000 });
  }
}
