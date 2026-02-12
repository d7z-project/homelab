import { Component, inject, OnInit, signal } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { ExampleService } from './generated';
import { MatSnackBar } from '@angular/material/snack-bar';
import { firstValueFrom } from 'rxjs';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [RouterOutlet],
  templateUrl: './app.component.html',
})
export class AppComponent implements OnInit {
  protected readonly title = signal('frontend');
  private exampleService = inject(ExampleService);
  private snackBar = inject(MatSnackBar);

  async ngOnInit() {
    try {
      const res = await firstValueFrom(this.exampleService.pingGet());
      console.log('Ping success:', res);
    } catch (err) {
      console.error('Ping failed:', err);
      this.snackBar.open('Ping 请求失败，请检查网络或后端状态', '关闭', { duration: 1000 });
    }
  }
}
