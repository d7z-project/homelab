import { Component } from '@angular/core';
import { MatDialogModule } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';

@Component({
  selector: 'app-logout-dialog',
  standalone: true,
  imports: [MatDialogModule, MatButtonModule],
  template: `
    <h2 mat-dialog-title>确认注销</h2>
    <mat-dialog-content>
      <p>确定要注销并退出系统吗？</p>
    </mat-dialog-content>
    <mat-dialog-actions align="end">
      <button mat-button mat-dialog-close>取消</button>
      <button mat-flat-button color="warn" [mat-dialog-close]="true">确认</button>
    </mat-dialog-actions>
  `,
})
export class LogoutDialogComponent {}
