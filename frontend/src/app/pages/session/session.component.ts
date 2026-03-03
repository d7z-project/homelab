import { Component, OnInit, inject, signal, computed } from '@angular/core';
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
  ],
  template: `
    <div class="animate-in fade-in duration-500 pb-20">
      @if (loading()) {
        <div class="fixed inset-0 bg-surface/40 z-50 flex items-center justify-center backdrop-blur-[1px]">
          <mat-spinner diameter="40"></mat-spinner>
        </div>
      }

      <div class="min-h-[calc(100vh-64px)] bg-surface-container-lowest py-6 px-4 sm:px-8">
        <div class="max-w-7xl mx-auto space-y-4">
          <!-- Data Info Bar -->
          <div class="flex items-center justify-between px-4 py-2 bg-surface-container-low/50 rounded-2xl border border-outline-variant/30 text-xs text-outline font-medium tracking-wide">
            <div class="flex items-center gap-4">
              <span class="flex items-center gap-1.5">
                <mat-icon class="!w-4 !h-4 !text-[14px]">admin_panel_settings</mat-icon>
                共 {{ sessions().length }} 个活跃会话
              </span>
            </div>
            <button mat-icon-button (click)="loadSessions()" [disabled]="loading()" matTooltip="刷新">
              <mat-icon class="!w-4 !h-4 !text-[14px]">refresh</mat-icon>
            </button>
          </div>

          <!-- Table Card -->
          <div class="bg-surface border border-outline-variant rounded-3xl overflow-hidden shadow-sm">
            <table mat-table [dataSource]="sessions()" class="w-full bg-transparent">
              <ng-container matColumnDef="ip">
                <th mat-header-cell *matHeaderCellDef class="!pl-6 bg-surface-container-low font-bold"> IP 地址 </th>
                <td mat-cell *matCellDef="let element" class="py-4 font-mono text-sm !pl-6 text-primary">
                  {{ element.ip || 'Unknown' }}
                </td>
              </ng-container>

              <ng-container matColumnDef="userAgent">
                <th mat-header-cell *matHeaderCellDef class="bg-surface-container-low font-bold"> 浏览器 / 设备 </th>
                <td mat-cell *matCellDef="let element" class="text-xs text-outline">
                  <span [matTooltip]="element.userAgent">{{ element.userAgent | ua }}</span>
                </td>
              </ng-container>

              <ng-container matColumnDef="createdAt">
                <th mat-header-cell *matHeaderCellDef class="bg-surface-container-low font-bold"> 创建时间 </th>
                <td mat-cell *matCellDef="let element" class="text-xs text-outline">
                  {{ element.createdAt | date: 'yyyy-MM-dd HH:mm:ss' }}
                </td>
              </ng-container>

              <ng-container matColumnDef="actions">
                <th mat-header-cell *matHeaderCellDef class="text-right !pr-6 bg-surface-container-low font-bold" style="width: 1px; white-space: nowrap"> 操作 </th>
                <td mat-cell *matCellDef="let element" class="text-right !pr-6" style="white-space: nowrap">
                  <button mat-icon-button color="warn" (click)="revokeSession(element)" 
                    [disabled]="element.id === uiService.sessionId()"
                    [matTooltip]="element.id === uiService.sessionId() ? '当前会话' : '吊销会话'">
                    <mat-icon class="!text-[20px] !w-5 !h-5 !flex !items-center !justify-center"
                      [class.opacity-20]="element.id === uiService.sessionId()">logout</mat-icon>
                  </button>
                </td>
              </ng-container>

              <tr mat-header-row *matHeaderRowDef="displayedColumns()"></tr>
              <tr mat-row *matRowDef="let row; columns: displayedColumns()"
                class="hover:bg-surface-container-low/50 transition-colors"
                [class.bg-primary/5]="row.id === uiService.sessionId()"></tr>

              <tr class="mat-mdc-row" *matNoDataRow>
                <td class="mat-mdc-cell py-12 text-center text-outline opacity-40 italic" [attr.colspan]="displayedColumns().length">
                  暂无活跃会话
                </td>
              </tr>
            </table>
          </div>
        </div>
      </div>
    </div>
  `,
})
export class SessionComponent implements OnInit {
  private authService = inject(AuthService);
  private snackBar = inject(MatSnackBar);
  private dialog = inject(MatDialog);
  private breakpointObserver = inject(BreakpointObserver);
  public uiService = inject(UiService);

  isHandset = toSignal(
    this.breakpointObserver.observe(Breakpoints.Handset).pipe(map((result) => result.matches)),
    { initialValue: this.breakpointObserver.isMatched(Breakpoints.Handset) }
  );

  loading = signal(false);
  sessions = signal<ModelsSession[]>([]);

  displayedColumns = computed(() =>
    this.isHandset() ? ['ip', 'actions'] : ['ip', 'userAgent', 'createdAt', 'actions']
  );

  ngOnInit(): void {
    this.uiService.configureToolbar({ shadow: true, sticky: true });
    this.loadSessions();
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
      this.snackBar.open('加载会话失败', '关闭', { duration: 3000 });
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
          message: `确定要吊销会话 "${session.ip || session.id}" 吗？该管理员将被强制下线。`,
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
