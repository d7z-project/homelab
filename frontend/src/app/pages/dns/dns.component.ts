import { Component, OnInit, inject, signal, computed, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { MatTableModule } from '@angular/material/table';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatCardModule } from '@angular/material/card';
import { MatTabsModule } from '@angular/material/tabs';
import { DnsService, ModelsDomain, ModelsRecord } from '../../generated';
import { MatDialog, MatDialogModule } from '@angular/material/dialog';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatMenuModule } from '@angular/material/menu';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatDividerModule } from '@angular/material/divider';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { firstValueFrom } from 'rxjs';
import { BreakpointObserver, Breakpoints } from '@angular/cdk/layout';
import { toSignal } from '@angular/core/rxjs-interop';
import { map } from 'rxjs/operators';
import { UiService } from '../../ui.service';
import { ConfirmDialogComponent } from '../rbac/confirm-dialog.component';
import { CreateDomainDialogComponent } from './create-domain-dialog.component';
import { CreateRecordDialogComponent } from './create-record-dialog.component';
import { PageHeaderComponent } from '../../shared/page-header.component';

@Component({
  selector: 'app-dns',
  standalone: true,
  imports: [
    CommonModule,
    MatTableModule,
    MatButtonModule,
    MatIconModule,
    MatCardModule,
    MatDialogModule,
    MatTabsModule,
    MatProgressBarModule,
    MatProgressSpinnerModule,
    MatTooltipModule,
    MatMenuModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatDividerModule,
    MatSlideToggleModule,
    PageHeaderComponent,
  ],
  templateUrl: './dns.component.html',
})
export class DnsComponent implements OnInit, OnDestroy {
  private dnsService = inject(DnsService);
  private snackBar = inject(MatSnackBar);
  private dialog = inject(MatDialog);
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private breakpointObserver = inject(BreakpointObserver);
  public uiService = inject(UiService);

  private scrollListener?: () => void;

  isHandset = toSignal(
    this.breakpointObserver.observe(Breakpoints.Handset).pipe(map((result) => result.matches)),
    { initialValue: this.breakpointObserver.isMatched(Breakpoints.Handset) },
  );

  domains = signal<ModelsDomain[]>([]);
  records = signal<ModelsRecord[]>([]);

  domainTotal = signal(0);
  recordTotal = signal(0);

  domainPage = signal(1);
  recordPage = signal(1);

  pageSize = signal(20);

  domainSearch = signal('');
  recordSearch = signal('');
  selectedDomainId = signal<string>('');

  loading = signal(false);
  loadingMore = signal(false);
  selectedTabIndex = signal(0);
  showScrollTop = signal(false);

  // Controlled signals for stability - Enabled column at START
  displayedDomainColumns = computed(() =>
    this.isHandset() ? ['enabled', 'name', 'actions'] : ['enabled', 'name', 'comments', 'updatedAt', 'actions'],
  );
  displayedRecordColumns = computed(() =>
    this.isHandset()
      ? ['enabled', 'name', 'type', 'actions']
      : ['enabled', 'name', 'type', 'value', 'ttl', 'actions'],
  );

  getDomainName(domainId: string | undefined): string {
    if (!domainId) return '未知域名';
    const domain = this.domains().find((d) => d.id === domainId);
    return domain?.name || '未知域名';
  }

  hasSearchContent = computed(() => {
    const tab = this.selectedTabIndex();
    if (tab === 0) return this.domainSearch().length > 0;
    if (tab === 1) return this.recordSearch().length > 0;
    return false;
  });

  openSearch() {
    const tab = this.selectedTabIndex();
    let placeholder = '';
    let value = '';
    let onSearch: (val: string) => void;

    if (tab === 0) {
      placeholder = '搜索域名或备注...';
      value = this.domainSearch();
      onSearch = (v) => this.onDomainSearch(v);
    } else {
      placeholder = '搜索主机记录或内容...';
      value = this.recordSearch();
      onSearch = (v) => this.onRecordSearch(v);
    }

    this.uiService.openSearch({
      placeholder,
      value,
      onSearch,
    });
  }

  currentSearchTerm = computed(() => {
    const tab = this.selectedTabIndex();
    if (tab === 0) return this.domainSearch();
    if (tab === 1) return this.recordSearch();
    return '';
  });

  hasMoreDomains = computed(() => this.domains().length < this.domainTotal());
  hasMoreRecords = computed(() => this.records().length < this.recordTotal());

  fabConfig = computed(() => {
    switch (this.selectedTabIndex()) {
      case 0:
        return {
          icon: 'add',
          label: '添加域名',
          action: () => this.createDomain(),
        };
      case 1:
        return {
          icon: 'add_link',
          label: '新增记录',
          action: () => this.createRecord(),
        };
      default:
        return null;
    }
  });

  constructor() {
    this.route.queryParams.subscribe((params) => {
      if (params['tab'] === 'domain') this.selectedTabIndex.set(0);
      else if (params['tab'] === 'record') {
        this.selectedTabIndex.set(1);
        if (params['domainId']) this.selectedDomainId.set(params['domainId']);
      }
    });
  }

  ngOnInit(): void {
    this.uiService.configureToolbar({ shadow: false });
    this.refreshAll();
    this.setupScrollListener();
  }

  ngOnDestroy(): void {
    this.uiService.resetToolbar();
    if (this.scrollListener) {
      const scrollElement = document.querySelector('mat-sidenav-content');
      scrollElement?.removeEventListener('scroll', this.scrollListener);
    }
  }

  private setupScrollListener() {
    const scrollElement = document.querySelector('mat-sidenav-content');
    if (!scrollElement) return;

    this.scrollListener = () => {
      this.showScrollTop.set(scrollElement.scrollTop > 300);
      const atBottom =
        scrollElement.scrollHeight - scrollElement.scrollTop <= scrollElement.clientHeight + 150;

      if (atBottom && !this.loadingMore() && !this.loading()) {
        const tab = this.selectedTabIndex();
        if (tab === 0 && this.hasMoreDomains()) {
          this.domainPage.update((p) => p + 1);
          this.loadMore('domain');
        } else if (tab === 1 && this.hasMoreRecords()) {
          this.recordPage.update((p) => p + 1);
          this.loadMore('record');
        }
      }
    };
    scrollElement.addEventListener('scroll', this.scrollListener);
  }

  scrollToTop() {
    const scrollElement = document.querySelector('mat-sidenav-content');
    if (scrollElement) {
      scrollElement.scrollTo({ top: 0, behavior: 'smooth' });
    }
  }

  private updateQueryParams() {
    const tabs = ['domain', 'record'];
    this.router.navigate([], {
      relativeTo: this.route,
      queryParams: {
        tab: tabs[this.selectedTabIndex()],
        domainId: this.selectedTabIndex() === 1 ? this.selectedDomainId() || null : null,
        domainSearch: this.domainSearch() || null,
        recordSearch: this.recordSearch() || null,
      },
      queryParamsHandling: 'merge',
      replaceUrl: true,
    });
  }

  onTabChange(index: number) {
    this.selectedTabIndex.set(index);
    this.updateQueryParams();

    // Refresh data for the selected tab
    if (index === 0) {
      this.domainPage.set(1);
      this.loadDomains(true);
    } else if (index === 1) {
      this.recordPage.set(1);
      this.loadRecords(true);
    }
  }

  async refreshAll() {
    this.loading.set(true);
    try {
      this.domainPage.set(1);
      this.recordPage.set(1);
      await Promise.all([this.loadDomains(true), this.loadRecords(true)]);
    } catch (err) {
      this.snackBar
        .open('加载失败', '重试')
        .onAction()
        .subscribe(() => this.refreshAll());
    } finally {
      this.loading.set(false);
    }
  }

  async loadDomains(reset = false) {
    const data = await firstValueFrom(
      this.dnsService.dnsDomainsGet(this.domainPage(), this.pageSize(), this.domainSearch()),
    );
    if (reset) this.domains.set(data.items || []);
    else {
      const current = this.domains();
      const newItems = (data.items || []).filter((n) => !current.some((e) => e.id === n.id));
      this.domains.update((prev) => [...prev, ...newItems]);
    }
    this.domainTotal.set(data.total || 0);
  }

  async loadRecords(reset = false) {
    const data = await firstValueFrom(
      this.dnsService.dnsRecordsGet(
        this.selectedDomainId(),
        this.recordPage(),
        this.pageSize(),
        this.recordSearch(),
      ),
    );
    if (reset) this.records.set(data.items || []);
    else {
      const current = this.records();
      const newItems = (data.items || []).filter((n) => !current.some((e) => e.id === n.id));
      this.records.update((prev) => [...prev, ...newItems]);
    }
    this.recordTotal.set(data.total || 0);
  }

  async loadMore(type: 'domain' | 'record') {
    this.loadingMore.set(true);
    try {
      if (type === 'domain') await this.loadDomains();
      else await this.loadRecords();
    } finally {
      this.loadingMore.set(false);
    }
  }

  onDomainSearch(term: string) {
    this.domainSearch.set(term);
    this.domainPage.set(1);
    this.loadDomains(true);
  }

  onRecordSearch(term: string) {
    this.recordSearch.set(term);
    this.recordPage.set(1);
    this.loadRecords(true);
  }

  onDomainFilterChange(domainId: string) {
    this.selectedDomainId.set(domainId);
    this.recordPage.set(1);
    this.updateQueryParams();
    this.loadRecords(true);
  }

  async toggleDomain(domain: ModelsDomain) {
    if (!domain.id) return;
    this.loading.set(true);
    try {
      const updated = { ...domain, enabled: !domain.enabled };
      await firstValueFrom(this.dnsService.dnsDomainsIdPut(domain.id, updated));
      this.snackBar.open(`域名已${updated.enabled ? '启用' : '禁用'}`, '关闭', { duration: 2000 });
      await this.loadDomains(true);
    } catch (err) {
      this.snackBar.open('操作失败', '关闭', { duration: 2000 });
    } finally {
      this.loading.set(false);
    }
  }

  async toggleRecord(record: ModelsRecord) {
    if (!record.id) return;
    this.loading.set(true);
    try {
      const updated = { ...record, enabled: !record.enabled };
      await firstValueFrom(this.dnsService.dnsRecordsIdPut(record.id, updated));
      this.snackBar.open(`记录已${updated.enabled ? '启用' : '禁用'}`, '关闭', { duration: 2000 });
      await this.loadRecords(true);
    } catch (err) {
      this.snackBar.open('操作失败', '关闭', { duration: 2000 });
    } finally {
      this.loading.set(false);
    }
  }

  createDomain() {
    requestAnimationFrame(() => {
      const dialogRef = this.dialog.open(CreateDomainDialogComponent, {
        data: { domain: null, existingNames: this.domains().map((d) => d.name || '') },
      });

      dialogRef.afterClosed().subscribe(async (result) => {
        if (result) {
          this.loading.set(true);
          try {
            await firstValueFrom(this.dnsService.dnsDomainsPost(result));
            this.snackBar.open('域名已创建', '关闭', { duration: 2000 });
            this.refreshAll();
          } catch (err: any) {
            this.snackBar.open('创建失败: ' + (err.error?.message || '未知错误'), '关闭');
          } finally {
            this.loading.set(false);
          }
        }
      });
    });
  }

  editDomain(domain: ModelsDomain) {
    requestAnimationFrame(() => {
      const dialogRef = this.dialog.open(CreateDomainDialogComponent, {
        data: { domain: domain, existingNames: this.domains().map((d) => d.name || '') },
      });

      dialogRef.afterClosed().subscribe(async (result) => {
        if (result && domain.id) {
          this.loading.set(true);
          try {
            await firstValueFrom(this.dnsService.dnsDomainsIdPut(domain.id, result));
            this.snackBar.open('域名配置已更新', '关闭', { duration: 2000 });
            this.refreshAll();
          } catch (err: any) {
            this.snackBar.open('更新失败: ' + (err.error?.message || '未知错误'), '关闭');
          } finally {
            this.loading.set(false);
          }
        }
      });
    });
  }

  deleteDomain(domain: ModelsDomain) {
    requestAnimationFrame(() => {
      const dialogRef = this.dialog.open(ConfirmDialogComponent, {
        data: {
          title: '删除域名',
          message: `确定要删除域名 "${domain.name}" 吗？此操作将级联删除所有关联的解析记录！`,
          confirmText: '确定删除',
          color: 'warn',
        },
      });

      dialogRef.afterClosed().subscribe(async (result) => {
        if (result && domain.id) {
          this.loading.set(true);
          try {
            await firstValueFrom(this.dnsService.dnsDomainsIdDelete(domain.id));
            this.snackBar.open('域名已删除', '关闭', { duration: 2000 });
            this.refreshAll();
          } catch (err) {
            this.snackBar.open('删除失败', '关闭', { duration: 2000 });
          } finally {
            this.loading.set(false);
          }
        }
      });
    });
  }

  createRecord() {
    requestAnimationFrame(() => {
      const dialogRef = this.dialog.open(CreateRecordDialogComponent, {
        data: { record: null, domains: this.domains(), defaultDomainId: this.selectedDomainId() },
      });

      dialogRef.afterClosed().subscribe(async (result) => {
        if (result) {
          this.loading.set(true);
          try {
            await firstValueFrom(this.dnsService.dnsRecordsPost(result));
            this.snackBar.open('解析记录已添加', '关闭', { duration: 2000 });
            this.loadRecords(true);
          } catch (err: any) {
            this.snackBar.open('添加失败: ' + (err.error?.message || '未知错误'), '关闭');
          } finally {
            this.loading.set(false);
          }
        }
      });
    });
  }

  editRecord(record: ModelsRecord) {
    requestAnimationFrame(() => {
      const dialogRef = this.dialog.open(CreateRecordDialogComponent, {
        data: { record: record, domains: this.domains() },
      });

      dialogRef.afterClosed().subscribe(async (result) => {
        if (result && record.id) {
          this.loading.set(true);
          try {
            await firstValueFrom(this.dnsService.dnsRecordsIdPut(record.id, result));
            this.snackBar.open('解析记录已更新', '关闭', { duration: 2000 });
            this.loadRecords(true);
          } catch (err: any) {
            this.snackBar.open('更新失败: ' + (err.error?.message || '未知错误'), '关闭');
          } finally {
            this.loading.set(false);
          }
        }
      });
    });
  }

  deleteRecord(record: ModelsRecord) {
    requestAnimationFrame(() => {
      const dialogRef = this.dialog.open(ConfirmDialogComponent, {
        data: {
          title: '删除记录',
          message: `确定要删除记录 "${record.name} (${record.type})" 吗？`,
          confirmText: '确定删除',
          color: 'warn',
        },
      });

      dialogRef.afterClosed().subscribe(async (result) => {
        if (result && record.id) {
          this.loading.set(true);
          try {
            await firstValueFrom(this.dnsService.dnsRecordsIdDelete(record.id));
            this.snackBar.open('记录已删除', '关闭', { duration: 2000 });
            this.loadRecords(true);
          } catch (err) {
            this.snackBar.open('删除失败', '关闭', { duration: 2000 });
          } finally {
            this.loading.set(false);
          }
        }
      });
    });
  }

  getRecordTypeClass(type: string): string {
    switch (type) {
      case 'A':
        return 'bg-primary-container text-on-primary-container';
      case 'AAAA':
        return 'bg-tertiary-container text-on-tertiary-container';
      case 'CNAME':
        return 'bg-warn-container text-on-warn-container';
      case 'MX':
        return 'bg-secondary-container text-on-secondary-container';
      case 'TXT':
        return 'bg-surface-container-high text-outline';
      default:
        return 'bg-surface-container-highest text-on-surface-variant';
    }
  }
}
