import { Component, inject } from '@angular/core';
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

  isHandset = toSignal(
    this.breakpointObserver.observe(Breakpoints.Handset).pipe(map((result) => result.matches)),
    { initialValue: false },
  );

  toggleDrawer() {
    this.uiService.toggleSidenav();
  }

  logout() {
    const dialogRef = this.dialog.open(LogoutDialogComponent, {
      width: '400px',
      maxWidth: '90vw',
    });

    dialogRef.afterClosed().subscribe((result) => {
      if (result) {
        this.authService.logoutPost().subscribe({
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
