import { Component, OnInit, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatTableModule } from '@angular/material/table';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatButtonModule } from '@angular/material/button';
import { MatTooltipModule } from '@angular/material/tooltip';
import { FormsModule } from '@angular/forms';
import { AuditService, AuditAuditLog } from '../../generated';
import { firstValueFrom } from 'rxjs';
import { MatSnackBar } from '@angular/material/snack-bar';

@Component({
  selector: 'app-audit',
  standalone: true,
  imports: [
    CommonModule,
    MatTableModule,
    MatIconModule,
    MatProgressSpinnerModule,
    MatButtonModule,
    MatTooltipModule,
    FormsModule,
  ],
  template: `
    <div class="animate-in fade-in duration-500 pb-20">
      @if (loading()) {
        <div class="fixed inset-0 bg-surface/40 z-50 flex items-center justify-center backdrop-blur-[1px]">
          <mat-spinner diameter="40"></mat-spinner>
        </div>
      }

      <!-- Floating Search Overlay -->
      @if (showSearch()) {
        <div 
          class="fixed inset-0 bg-black/30 backdrop-blur-[1px] z-[90] animate-in fade-in duration-300"
          (click)="showSearch.set(false)"
        ></div>

        <div class="fixed top-4 left-0 right-0 z-[100] px-4 flex justify-center animate-in slide-in-from-top-4 duration-300">
          <div class="w-full max-w-2xl bg-surface border border-outline-variant rounded-full px-6 py-2 flex items-center gap-4 shadow-2xl ring-1 ring-black/5 backdrop-blur-md bg-surface/95">
            <mat-icon class="text-primary">search</mat-icon>
            <input
              class="flex-1 bg-transparent border-none outline-none text-on-surface placeholder:text-outline/60 py-2 text-lg"
              [(ngModel)]="search"
              (input)="onSearchChange($event)"
              placeholder="搜索操作人、动作、资源或目标 ID..."
              autofocus
            />
            <button mat-icon-button (click)="showSearch.set(false)">
              <mat-icon>close</mat-icon>
            </button>
          </div>
        </div>
      }

      <div class="min-h-[calc(100vh-64px)] bg-surface-container-lowest py-8 px-4 sm:px-8">
        <div class="max-w-7xl mx-auto space-y-4">
          
          <!-- Data Info Bar -->
          <div class="flex items-center justify-between px-4 py-2 bg-surface-container-low/50 rounded-2xl border border-outline-variant/30 text-xs text-outline font-medium tracking-wide">
            <div class="flex items-center gap-4">
              <span class="flex items-center gap-1.5"><mat-icon class="!w-4 !h-4 !text-[14px]">history</mat-icon> 共 {{ total() }} 条日志</span>
              @if (search()) {
                <div class="flex items-center gap-1.5 text-primary bg-primary/10 pl-3 pr-1.5 py-1 rounded-full border border-primary/20 animate-in zoom-in-95 group shadow-sm">
                  <mat-icon class="!w-4 !h-4 !text-[14px]">filter_alt</mat-icon>
                  <span>正在搜索: "{{ search() }}"</span>
                  <button class="w-5 h-5 flex items-center justify-center rounded-full bg-primary/20 text-primary hover:bg-primary hover:text-on-primary transition-all duration-200 ml-1 cursor-pointer border-none p-0" (click)="clearSearch()" title="清除搜索">
                    <mat-icon class="!w-3 !h-3 !text-[12px] font-bold">close</mat-icon>
                  </button>
                </div>
              }
            </div>
            <span class="opacity-60">已加载 {{ logs().length }}</span>
          </div>

          <div class="bg-surface border border-outline-variant rounded-3xl overflow-hidden shadow-sm">
            <table mat-table [dataSource]="logs()" class="w-full bg-transparent">
              
              <ng-container matColumnDef="timestamp">
                <th mat-header-cell *matHeaderCellDef class="!pl-6 bg-surface-container-low font-bold">时间</th>
                <td mat-cell *matCellDef="let element" class="py-4 text-xs font-mono text-outline !pl-6 whitespace-nowrap">
                  {{ element.timestamp | date:'yyyy-MM-dd HH:mm:ss' }}
                </td>
              </ng-container>

              <ng-container matColumnDef="subject">
                <th mat-header-cell *matHeaderCellDef class="bg-surface-container-low font-bold">操作人</th>
                <td mat-cell *matCellDef="let element" class="py-4 font-medium text-primary">
                  {{ element.subject }}
                </td>
              </ng-container>

              <ng-container matColumnDef="action">
                <th mat-header-cell *matHeaderCellDef class="bg-surface-container-low font-bold">动作</th>
                <td mat-cell *matCellDef="let element">
                  <span class="px-2 py-0.5 rounded-full text-[10px] font-bold tracking-wider"
                        [ngClass]="{
                          'bg-primary-container text-on-primary-container': element.action === 'POST',
                          'bg-tertiary-container text-on-tertiary-container': element.action === 'PUT',
                          'bg-error/20 text-error': element.action === 'DELETE'
                        }">
                    {{ element.action }}
                  </span>
                </td>
              </ng-container>

              <ng-container matColumnDef="resource">
                <th mat-header-cell *matHeaderCellDef class="bg-surface-container-low font-bold">资源模块</th>
                <td mat-cell *matCellDef="let element" class="text-sm font-medium">
                  {{ element.resource }}
                </td>
              </ng-container>

              <ng-container matColumnDef="targetId">
                <th mat-header-cell *matHeaderCellDef class="bg-surface-container-low font-bold">目标 ID</th>
                <td mat-cell *matCellDef="let element" class="text-xs font-mono opacity-80 max-w-[150px] truncate" [matTooltip]="element.targetId">
                  {{ element.targetId || '-' }}
                </td>
              </ng-container>

              <ng-container matColumnDef="status">
                <th mat-header-cell *matHeaderCellDef class="text-right !pr-6 bg-surface-container-low font-bold">状态</th>
                <td mat-cell *matCellDef="let element" class="text-right !pr-6">
                  <mat-icon 
                    class="!text-[18px] !w-[18px] !h-[18px] !flex !items-center !justify-center"
                    [class.text-green-500]="element.status === 'Success'"
                    [class.text-error]="element.status !== 'Success'"
                    [matTooltip]="element.status"
                  >
                    {{ element.status === 'Success' ? 'check_circle' : 'error' }}
                  </mat-icon>
                </td>
              </ng-container>

              <tr mat-header-row *matHeaderRowDef="columns"></tr>
              <tr mat-row *matRowDef="let row; columns: columns" class="hover:bg-surface-container-low/50 transition-colors"></tr>
              
              <tr class="mat-mdc-row" *matNoDataRow>
                <td class="mat-mdc-cell py-16 text-center text-outline opacity-40 italic" [attr.colspan]="columns.length">
                  {{ search() ? '没有找到符合条件的审计记录' : '暂无审计记录' }}
                </td>
              </tr>
            </table>
          </div>

          @if (loadingMore()) {
            <div class="py-10 flex justify-center">
              <mat-spinner diameter="32"></mat-spinner>
            </div>
          }
          
          <div class="flex justify-center pb-12 pt-4" *ngIf="hasMore()">
             <button mat-fab extended color="primary" class="!rounded-2xl" (click)="loadMore()" [disabled]="loadingMore()">
               <mat-icon>expand_more</mat-icon>
               加载更多记录
             </button>
          </div>

        </div>
      </div>

      <!-- Search FAB -->
      <div class="fixed bottom-8 right-8 z-50 flex flex-col gap-4 items-end">
        <button 
          mat-fab 
          [color]="search() ? 'tertiary' : 'secondary'"
          (click)="showSearch() ? (search() ? clearSearch() : showSearch.set(false)) : showSearch.set(true)"
          class="!rounded-2xl shadow-lg hover:shadow-xl transition-all duration-300 animate-in slide-in-from-bottom-2"
          [matTooltip]="search() ? (showSearch() ? '清除搜索' : '正在筛选') : (showSearch() ? '关闭' : '搜索')"
        >
          <mat-icon>{{ showSearch() ? (search() ? 'filter_alt_off' : 'close') : (search() ? 'filter_alt' : 'search') }}</mat-icon>
        </button>
      </div>
    </div>
  `
})
export class AuditComponent implements OnInit {
  private auditService = inject(AuditService);
  private snackBar = inject(MatSnackBar);

  logs = signal<AuditAuditLog[]>([]);
  total = signal(0);
  page = signal(0);
  pageSize = signal(50);
  loading = signal(false);
  loadingMore = signal(false);
  
  showSearch = signal(false);
  search = signal('');

  columns: string[] = ['timestamp', 'subject', 'action', 'resource', 'targetId', 'status'];

  hasMore = computed(() => this.logs().length < this.total());

  ngOnInit() {
    this.refresh();
  }

  onSearchChange(event: any) {
    // Basic debounce could be added here if needed, but for local-like homelab speed, direct is fine
    this.refresh();
  }

  clearSearch() {
    this.search.set('');
    this.showSearch.set(false);
    this.refresh();
  }

  async refresh() {
    this.loading.set(true);
    this.page.set(0);
    try {
      const data = await firstValueFrom(this.auditService.auditLogsGet(this.page(), this.pageSize(), this.search()));
      this.logs.set(data.items || []);
      this.total.set(data.total || 0);
    } catch (err) {
      this.snackBar.open('加载日志失败', '关闭', { duration: 3000 });
    } finally {
      this.loading.set(false);
    }
  }

  async loadMore() {
    if (!this.hasMore()) return;
    this.loadingMore.set(true);
    this.page.update(p => p + 1);
    try {
      const data = await firstValueFrom(this.auditService.auditLogsGet(this.page(), this.pageSize(), this.search()));
      this.logs.update(prev => [...prev, ...(data.items || [])]);
    } catch (err) {
      this.page.update(p => p - 1);
      this.snackBar.open('加载更多失败', '关闭', { duration: 3000 });
    } finally {
      this.loadingMore.set(false);
    }
  }
}
