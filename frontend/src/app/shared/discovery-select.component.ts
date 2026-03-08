import {
  Component,
  Input,
  OnInit,
  inject,
  signal,
  ViewChild,
  ElementRef,
  Optional,
  Self,
  computed,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  ControlValueAccessor,
  FormsModule,
  ReactiveFormsModule,
  FormControl,
  NgControl,
  Validator,
  AbstractControl,
  ValidationErrors,
  FormGroupDirective,
  NgForm,
} from '@angular/forms';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import {
  MatAutocompleteModule,
  MatAutocompleteSelectedEvent,
} from '@angular/material/autocomplete';
import { MatChipsModule } from '@angular/material/chips';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { ErrorStateMatcher } from '@angular/material/core';
import { DiscoveryService, ModelsLookupItem } from '../generated';
import {
  debounceTime,
  distinctUntilChanged,
  switchMap,
  of,
  catchError,
  finalize,
  forkJoin,
} from 'rxjs';

/**
 * Custom error state matcher to link internal input error state to external ngControl
 */
class CrossFieldMatcher implements ErrorStateMatcher {
  constructor(private parentControl: NgControl | null) {}
  isErrorState(control: FormControl | null, form: FormGroupDirective | NgForm | null): boolean {
    if (!this.parentControl) return false;
    return !!(
      this.parentControl.invalid &&
      (this.parentControl.touched || this.parentControl.dirty)
    );
  }
}

@Component({
  selector: 'app-discovery-select',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    ReactiveFormsModule,
    MatFormFieldModule,
    MatInputModule,
    MatAutocompleteModule,
    MatChipsModule,
    MatIconModule,
    MatProgressSpinnerModule,
    MatProgressBarModule,
  ],
  template: `
    <mat-form-field
      [appearance]="appearance"
      class="w-full m3-form-field"
      [subscriptSizing]="subscriptSizing"
    >
      <mat-label class="font-bold text-xs uppercase tracking-widest">{{ label }}</mat-label>

      @if (multiple) {
        <mat-chip-grid #chipGrid [errorStateMatcher]="matcher">
          @for (item of selectedItems(); track item.id) {
            <mat-chip-row (removed)="removeItem(item)" class="m3-chip">
              <div class="flex flex-col leading-tight py-0.5">
                <span class="text-[10px] font-black">{{ item.name }}</span>
                @if (item.description) {
                  <span class="text-[8px] opacity-60 truncate max-w-[120px]">{{
                    item.description
                  }}</span>
                }
              </div>
              <button matChipRemove><mat-icon>cancel</mat-icon></button>
            </mat-chip-row>
          }
          <input
            [placeholder]="placeholder"
            [matAutocomplete]="auto"
            [matChipInputFor]="chipGrid"
            [formControl]="inputControl"
            (blur)="onTouched()"
            #inputElement
          />
        </mat-chip-grid>
      } @else {
        <input
          matInput
          [placeholder]="placeholder"
          [matAutocomplete]="auto"
          [formControl]="inputControl"
          [errorStateMatcher]="matcher"
          (blur)="onTouched()"
          #inputElement
          class="font-medium"
        />
      }

      <mat-icon
        matSuffix
        class="text-primary opacity-40 transition-opacity group-hover:opacity-100"
      >
        expand_more
      </mat-icon>

      <!-- M3 Loading Indicator -->
      <div
        class="absolute left-0 right-0 bottom-0 overflow-hidden transition-all duration-500 pointer-events-none"
        [style.height]="isLoading() ? '3px' : '0px'"
      >
        <mat-progress-bar mode="indeterminate" class="m3-loader"></mat-progress-bar>
      </div>

      <mat-autocomplete
        #auto="matAutocomplete"
        [displayWith]="displayFn"
        (optionSelected)="onSelected($event)"
        class="m3-autocomplete-panel"
      >
        @if (showAllOption) {
          <mat-option [value]="{ id: '', name: allOptionLabel }" class="m3-option">
            <div class="flex items-center gap-4">
              <div class="m3-option-icon bg-primary/10 text-primary">
                <mat-icon>all_inclusive</mat-icon>
              </div>
              <span class="font-black text-sm">{{ allOptionLabel }}</span>
            </div>
          </mat-option>
        }
        @for (item of items(); track item.id) {
          <mat-option [value]="item" class="m3-option">
            <div class="flex items-start gap-4">
              <div class="m3-option-icon bg-surface-container text-outline/60">
                <mat-icon>{{ item.icon || 'label' }}</mat-icon>
              </div>
              <div class="flex flex-col min-w-0 flex-1 leading-tight py-1">
                <span class="font-black text-sm text-on-surface truncate">{{ item.name }}</span>
                <span class="text-[10px] font-mono text-outline truncate opacity-50">
                  {{ item.id }}
                </span>
                @if (item.description) {
                  <span class="text-[11px] text-outline truncate opacity-80 mt-1 italic">
                    {{ item.description }}
                  </span>
                }
              </div>
            </div>
          </mat-option>
        }
        @if (items().length === 0 && !isLoading() && lastSearch()) {
          <mat-option disabled class="text-xs italic opacity-50">未找到匹配项</mat-option>
        }
      </mat-autocomplete>

      @if (hint && (!ngControl || !ngControl.invalid)) {
        <mat-hint class="text-[10px] font-medium opacity-60">{{ hint }}</mat-hint>
      }
    </mat-form-field>
  `,
  styles: [
    `
      :host {
        display: block;
      }
      ::ng-deep .m3-form-field .mat-mdc-form-field-wrapper {
        padding-bottom: 0;
      }
      ::ng-deep .m3-form-field .mat-mdc-text-field-wrapper {
        border-radius: 16px !important; /* M3 extra rounded */
        background-color: var(--mat-sys-surface-container-low) !important;
        transition: all 0.2s ease-in-out;
      }
      ::ng-deep .m3-form-field.mat-form-field-invalid .mat-mdc-text-field-wrapper {
        background-color: var(--mat-sys-error-container) !important;
        opacity: 0.8;
      }
      ::ng-deep .m3-form-field .mat-mdc-form-field-focus-overlay {
        background-color: transparent !important;
      }
      ::ng-deep .m3-form-field .mat-mdc-form-field-label {
        color: var(--mat-sys-outline) !important;
      }
      ::ng-deep .m3-form-field.mat-focused .mat-mdc-form-field-label {
        color: var(--mat-sys-primary) !important;
      }

      .m3-chip {
        border-radius: 12px !important;
        background-color: var(--mat-sys-secondary-container) !important;
        color: var(--mat-sys-on-secondary-container) !important;
        border: none !important;
      }

      .m3-option {
        height: auto !important;
        min-height: 64px !important;
        margin: 4px 8px !important;
        border-radius: 16px !important;
        transition: background-color 0.2s;
      }
      .m3-option-icon {
        width: 40px;
        height: 40px;
        border-radius: 12px;
        display: flex;
        align-items: center;
        justify-content: center;
        flex-shrink: 0;
      }
      .m3-option-icon mat-icon {
        font-size: 20px;
        width: 20px;
        height: 20px;
      }

      ::ng-deep .m3-autocomplete-panel {
        border-radius: 24px !important;
        margin-top: 8px !important;
        padding: 8px 0 !important;
        box-shadow: var(--shadow-lg) !important;
        background-color: var(--mat-sys-surface-container-high) !important;
        border: 1px solid var(--mat-sys-outline-variant) !important;
      }

      .m3-loader {
        height: 3px !important;
        background-color: transparent !important;
      }
      ::ng-deep .m3-loader .mat-mdc-progress-bar-background {
        display: none;
      }
    `,
  ],
})
export class DiscoverySelectComponent implements OnInit, ControlValueAccessor, Validator {
  private discoveryService = inject(DiscoveryService);

  @Input() code = '';
  @Input() label = '选择';
  @Input() placeholder = '搜索...';
  @Input() hint = '';
  @Input() appearance: 'fill' | 'outline' = 'outline';
  @Input() subscriptSizing: 'fixed' | 'dynamic' = 'fixed';
  @Input() multiple = false;
  @Input() showAllOption = false;
  @Input() allOptionLabel = '全部';

  @ViewChild('inputElement') inputElement!: ElementRef<HTMLInputElement>;

  inputControl = new FormControl('');
  items = signal<ModelsLookupItem[]>([]);
  selectedItems = signal<ModelsLookupItem[]>([]);
  isLoading = signal(false);
  lastSearch = signal('');
  disabled = false;

  matcher: ErrorStateMatcher;

  constructor(@Optional() @Self() public ngControl: NgControl) {
    if (this.ngControl) {
      this.ngControl.valueAccessor = this;
    }
    this.matcher = new CrossFieldMatcher(this.ngControl);
  }

  public onChange: (value: any) => void = () => {};
  public onTouched: () => void = () => {};

  ngOnInit() {
    if (this.ngControl && this.ngControl.control) {
      this.ngControl.control.addValidators(this.validate.bind(this));
    }

    this.inputControl.valueChanges
      .pipe(
        debounceTime(300),
        distinctUntilChanged(),
        switchMap((search) => {
          if (typeof search !== 'string') return of(null);
          if (!this.code) return of({ items: [], total: 0 });
          this.isLoading.set(true);
          this.lastSearch.set(search);
          return this.discoveryService.discoveryLookupGet(this.code, search, '', 20).pipe(
            catchError(() => of({ items: [], total: 0 })),
            finalize(() => this.isLoading.set(false)),
          );
        }),
      )
      .subscribe((res) => {
        if (res) this.items.set(res.items || []);
      });

    if (this.code) {
      this.discoveryService
        .discoveryLookupGet(this.code, '', '', 20)
        .subscribe((res) => this.items.set(res.items || []));
    }
  }

  displayFn(item: any): string {
    if (typeof item === 'string') return item;
    return item ? item.name || '' : '';
  }

  onSelected(event: MatAutocompleteSelectedEvent) {
    const item = event.option.value as ModelsLookupItem;
    if (this.multiple) {
      const current = this.selectedItems();
      if (!current.find((i) => i.id === item.id)) {
        this.selectedItems.set([...current, item]);
        this.triggerChange();
      }
      this.inputControl.setValue('', { emitEvent: true });
    } else {
      this.selectedItems.set([item]);
      this.inputControl.setValue(item.name || '', { emitEvent: false });
      this.triggerChange();
    }
    this.onTouched();
  }

  removeItem(item: ModelsLookupItem) {
    this.selectedItems.set(this.selectedItems().filter((i) => i.id !== item.id));
    this.triggerChange();
  }

  private triggerChange() {
    const val = this.multiple
      ? this.selectedItems().map((i) => i.id)
      : this.selectedItems()[0]?.id || '';
    this.onChange(val);
  }

  validate(control: AbstractControl): ValidationErrors | null {
    return null;
  }

  writeValue(value: any): void {
    if (value === '' && this.showAllOption) {
      this.selectedItems.set([{ id: '', name: this.allOptionLabel }]);
      this.inputControl.setValue(this.allOptionLabel, { emitEvent: false });
      return;
    }

    if (!value || (Array.isArray(value) && value.length === 0)) {
      this.selectedItems.set([]);
      this.inputControl.setValue('', { emitEvent: false });
      return;
    }

    const ids = Array.isArray(value) ? value : [value];
    const currentIds = this.selectedItems().map((i) => i.id);

    if (JSON.stringify(ids) === JSON.stringify(currentIds)) {
      return;
    }

    const found = this.items().filter((item) => ids.includes(item.id));
    if (found.length === ids.length) {
      this.selectedItems.set(found);
      if (!this.multiple) {
        this.inputControl.setValue(found[0]?.name || '', { emitEvent: false });
      }
      return;
    }

    if (!this.code) return;
    this.isLoading.set(true);
    const lookups = ids.map((id) =>
      this.discoveryService.discoveryLookupGet(this.code, id, '', 1).pipe(
        catchError(() => of({ items: [] })),
        switchMap((res) => {
          const item = res.items?.find((i) => i.id === id);
          return of(item);
        }),
      ),
    );

    forkJoin(lookups).subscribe((items) => {
      const validItems = items.filter((i): i is ModelsLookupItem => !!i);
      this.selectedItems.set(validItems);
      if (!this.multiple && validItems.length > 0) {
        this.inputControl.setValue(validItems[0].name || '', { emitEvent: false });
      }
      this.isLoading.set(false);
    });
  }

  registerOnChange(fn: any): void {
    this.onChange = fn;
  }
  registerOnTouched(fn: any): void {
    this.onTouched = fn;
  }
  setDisabledState(isDisabled: boolean): void {
    this.disabled = isDisabled;
    if (isDisabled) this.inputControl.disable();
    else this.inputControl.enable();
  }
}
