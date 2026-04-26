import { Component, inject, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatCardModule } from '@angular/material/card';
import { MatGridListModule } from '@angular/material/grid-list';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { MatDialog } from '@angular/material/dialog';
import { Router } from '@angular/router';
import { AuthService } from '../../generated';
import { LogoutDialogComponent } from '../main/logout-dialog.component';
import { MatTooltipModule } from '@angular/material/tooltip';
import { UiService } from '../../ui.service';
import { BreakpointObserver, Breakpoints } from '@angular/cdk/layout';
import { toSignal } from '@angular/core/rxjs-interop';
import { map } from 'rxjs/operators';

@Component({
  selector: 'app-welcome',
  standalone: true,
  imports: [
    CommonModule,
    MatCardModule,
    MatGridListModule,
    MatIconModule,
    MatButtonModule,
    MatTooltipModule,
  ],
  templateUrl: './welcome.component.html',
})
export class WelcomeComponent {
  private dialog = inject(MatDialog);
  private authService = inject(AuthService);
  private router = inject(Router);
  private uiService = inject(UiService);
  private breakpointObserver = inject(BreakpointObserver);

  // 模拟图表点位 (0-100)
  chartPoints = signal<number[]>([30, 45, 35, 60, 55, 70, 65, 80, 75, 90, 85, 95]);

  // 生成 SVG 路径
  chartPath = computed(() => {
    const points = this.chartPoints();
    const width = 1000;
    const height = 200;
    const step = width / (points.length - 1);

    return points.map((p, i) => `${i * step},${height - (p / 100) * height}`).join(' L ');
  });

  // 生成面积路径
  areaPath = computed(() => {
    const path = this.chartPath();
    return `M 0,200 L ${path} L 1000,200 Z`;
  });

  isHandset = toSignal(
    this.breakpointObserver.observe(Breakpoints.Handset).pipe(map((result) => result.matches)),
    { initialValue: this.breakpointObserver.isMatched(Breakpoints.Handset) },
  );

  toggleDrawer() {
    this.uiService.toggleSidenav();
  }

  logout() {
    const dialogRef = this.dialog.open(LogoutDialogComponent, {
      maxWidth: '90vw',
    });

    dialogRef.afterClosed().subscribe((result) => {
      if (result) {
        this.authService.authLogoutPost().subscribe({
          next: () => {
            localStorage.clear();
            this.router.navigate(['/login']);
          },
          error: () => {
            localStorage.clear();
            this.router.navigate(['/login']);
          },
        });
      }
    });
  }
}
