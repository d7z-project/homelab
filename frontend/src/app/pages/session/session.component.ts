import { Component, OnInit, inject, signal, computed, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatTableModule } from '@angular/material/table';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { AuthService, ModelsSession } from '../../generated';
import { MatDialog, MatDialogModule } from '@angular/material/dialog';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatTooltipModule } from '@angular/material/tooltip';
import { firstValueFrom } from 'rxjs';
import { ConfirmDialogComponent } from '../rbac/confirm-dialog.component';
import { BreakpointObserver, Breakpoints } from '@angular/cdk/layout';
import { toSignal } from '@angular/core/rxjs-interop';
import { map } from 'rxjs/operators';
import { UiService } from '../../ui.service';
import { UaPipe } from '../../ua.pipe';
import { PageHeaderComponent } from '../../shared/page-header.component';

@Component({
  selector: 'app-session',
  standalone: true,
  imports: [
    CommonModule,
    MatTableModule,
    MatButtonModule,
    MatIconModule,
    MatDialogModule,
    MatProgressSpinnerModule,
    MatTooltipModule,
    UaPipe,
    PageHeaderComponent,
  ],
  template: `
    <div class="animate-in fade-in duration-500 pb-20">
      @if (loading() && sessions().length === 0) {
        <div
          class="fixed inset-0 bg-surface/40 z-50 flex items-center justify-center backdrop-blur-[1px]"
        >
          <mat-spinner diameter="40"></mat-spinner>
        </div>
      }

      <div class="min-h-[calc(100vh-64px)] bg-surface-container-lowest py-8 px-4 sm:px-8">
        <div class="max-w-7xl mx-auto space-y-6">
          <!-- Header Area -->
          <app-page-header
            title="管理会话"
            subtitle="监控并管理所有活跃的管理员登录会话"
            icon="admin_panel_settings"
            [total]="sessions().length"
            [loading]="loading()"
            unit="个活跃会话"
            (refresh)="loadSessions()"
          ></app-page-header>

          <!-- Table Card -->
          <div
            class="bg-surface border border-outline-variant rounded-[32px] overflow-hidden shadow-sm transition-shadow hover:shadow-md"
          >
            <table mat-table [dataSource]="sessions()" class="w-full bg-transparent">
              <!-- IP Column -->
              <ng-container matColumnDef="ip">
                <th
                  mat-header-cell
                  *matHeaderCellDef
                  class="!pl-8 bg-surface-container-low font-bold text-on-surface-variant uppercase tracking-wider text-[11px]"
                >
                  IP 地址
                </th>
                <td
                  mat-cell
                  *matCellDef="let element"
                  class="!pl-8 py-4 font-mono text-sm text-primary"
                >
                  {{ element.ip || 'Unknown' }}
                  @if (element.id === uiService.sessionId()) {
                    <span
                      class="ml-2 px-1.5 py-0.5 rounded bg-primary/10 text-primary text-[9px] font-bold uppercase"
                      >Current</span
                    >
                  }
                </td>
              </ng-container>

              <!-- Browser Column -->
              <ng-container matColumnDef="userAgent">
                <th
                  mat-header-cell
                  *matHeaderCellDef
                  class="bg-surface-container-low font-bold text-on-surface-variant uppercase tracking-wider text-[11px]"
                >
                  浏览器 / 设备
                </th>
                <td mat-cell *matCellDef="let element" class="py-4">
                  <span
                    class="px-2.5 py-1 rounded-lg bg-surface-container-high text-on-surface text-xs font-medium border border-outline-variant/50"
                    [matTooltip]="element.userAgent"
                  >
                    {{ element.userAgent | ua }}
                  </span>
                </td>
              </ng-container>

              <!-- CreatedAt Column -->
              <ng-container matColumnDef="createdAt">
                <th
                  mat-header-cell
                  *matHeaderCellDef
                  class="bg-surface-container-low font-bold text-on-surface-variant uppercase tracking-wider text-[11px]"
                >
                  创建时间
                </th>
                <td mat-cell *matCellDef="let element" class="py-4 text-xs font-mono text-outline">
                  {{ element.createdAt | date: 'yyyy-MM-dd HH:mm:ss' }}
                </td>
              </ng-container>

              <!-- Actions Column -->
              <ng-container matColumnDef="actions">
                <th
                  mat-header-cell
                  *matHeaderCellDef
                  class="!pr-8 bg-surface-container-low text-right"
                ></th>
                <td mat-cell *matCellDef="let element" class="!pr-8 text-right">
                  <button
                    mat-icon-button
                    color="warn"
                    (click)="revokeSession(element)"
                    [disabled]="element.id === uiService.sessionId()"
                    [matTooltip]="
                      element.id === uiService.sessionId() ? '当前会话不可吊销' : '吊销此会话'
                    "
                    class="hover:bg-error/10 transition-colors"
                  >
                    <mat-icon
                      class="!text-[20px] !w-5 !h-5 !flex !items-center !justify-center"
                      [class.opacity-20]="element.id === uiService.sessionId()"
                      >logout</mat-icon
                    >
                  </button>
                </td>
              </ng-container>

              <tr mat-header-row *matHeaderRowDef="displayedColumns()"></tr>
              <tr
                mat-row
                *matRowDef="let row; columns: displayedColumns()"
                class="hover:bg-surface-container-low/50 transition-colors"
                [class.bg-primary/5]="row.id === uiService.sessionId()"
              ></tr>

              <!-- Empty State -->
              <tr class="mat-mdc-row" *matNoDataRow>
                <td
                  class="mat-mdc-cell py-24 text-center"
                  [attr.colspan]="displayedColumns().length"
                >
                  <div class="flex flex-col items-center gap-3 opacity-30">
                    <mat-icon class="!w-16 !h-16 !text-[64px]">no_accounts</mat-icon>
                    <span class="text-lg font-medium italic">暂无活跃会话记录</span>
                  </div>
                </td>
              </tr>
            </table>
          </div>
        </div>
      </div>

      <!-- FAB for Scroll Top -->
      @if (showScrollTop()) {
        <div class="fixed bottom-8 right-8 z-50 animate-in slide-in-from-bottom-4">
          <button mat-fab color="tertiary" (click)="scrollToTop()" class="!rounded-2xl shadow-lg">
            <mat-icon>arrow_upward</mat-icon>
          </button>
        </div>
      }
    </div>
  `,
})
export class SessionComponent implements OnInit, OnDestroy {
  private authService = inject(AuthService);
  private snackBar = inject(MatSnackBar);
  private dialog = inject(MatDialog);
  private breakpointObserver = inject(BreakpointObserver);
  public uiService = inject(UiService);

  private scrollListener?: () => void;

  isHandset = toSignal(
    this.breakpointObserver.observe(Breakpoints.Handset).pipe(map((result) => result.matches)),
    { initialValue: this.breakpointObserver.isMatched(Breakpoints.Handset) },
  );

  loading = signal(false);
  sessions = signal<ModelsSession[]>([]);
  showScrollTop = signal(false);

  displayedColumns = computed(() =>
    this.isHandset() ? ['ip', 'actions'] : ['ip', 'userAgent', 'createdAt', 'actions'],
  );

  ngOnInit(): void {
    this.uiService.configureToolbar({ shadow: true, sticky: true });
    this.loadSessions();
    this.setupScrollListener();
  }

  ngOnDestroy(): void {
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
    };
    scrollElement.addEventListener('scroll', this.scrollListener);
  }

  scrollToTop() {
    const scrollElement = document.querySelector('mat-sidenav-content');
    if (scrollElement) {
      scrollElement.scrollTo({ top: 0, behavior: 'smooth' });
    }
  }

  async loadSessions() {
    this.loading.set(true);
    try {
      const data = await firstValueFrom(this.authService.authSessionsGet());
      const currentId = this.uiService.sessionId();
      let sorted = data || [];
      if (currentId) {
        sorted = [...sorted].sort((a, b) => {
          if (a.id === currentId) return -1;
          if (b.id === currentId) return 1;
          return 0;
        });
      }
      this.sessions.set(sorted);
    } catch (err) {
      this.snackBar
        .open('加载会话失败', '重试', { duration: 3000 })
        .onAction()
        .subscribe(() => this.loadSessions());
    } finally {
      this.loading.set(false);
    }
  }

  async revokeSession(session: ModelsSession) {
    if (!session.id) return;

    requestAnimationFrame(() => {
      const dialogRef = this.dialog.open(ConfirmDialogComponent, {
        data: {
          title: '吊销会话',
          message: `确定要吊销来自 IP "${session.ip || '未知'}" 的会话吗？该管理员将被强制下线。`,
          confirmText: '确定吊销',
          color: 'warn',
        },
      });

      dialogRef.afterClosed().subscribe(async (result) => {
        if (result && session.id) {
          this.loading.set(true);
          try {
            await firstValueFrom(this.authService.authSessionsIdDelete(session.id));
            this.snackBar.open('会话已吊销', '关闭', { duration: 2000 });
            await this.loadSessions();
          } catch (err) {
            this.snackBar.open('操作失败', '关闭', { duration: 2000 });
          } finally {
            this.loading.set(false);
          }
        }
      });
    });
  }
}
