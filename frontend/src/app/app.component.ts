import { Component, inject, OnInit, signal } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { AuthService } from './generated';
import { UiService } from './ui.service';
import { finalize } from 'rxjs';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [RouterOutlet, MatProgressSpinnerModule],
  templateUrl: './app.component.html',
})
export class AppComponent implements OnInit {
  private readonly authService = inject(AuthService);
  protected readonly uiService = inject(UiService);

  ngOnInit() {
    this.authService
      .infoGet()
      .pipe(finalize(() => this.uiService.initializing.set(false)))
      .subscribe({
        next: (info) => {
          console.log('App initialized with user:', info);
          this.uiService.userType.set(info.type ?? null);
          this.uiService.sessionId.set(info.sessionId ?? null);
        },
        error: (err) => {
          console.warn('App initialized with error (likely unauthenticated):', err);
          this.uiService.userType.set(null);
          this.uiService.sessionId.set(null);
        },
      });
  }
}
