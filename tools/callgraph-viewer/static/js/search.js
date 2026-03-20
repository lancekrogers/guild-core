// Search component for filtering graphs
export class Search {
    constructor(inputElement, navigation) {
        this.input = inputElement;
        this.navigation = navigation;
        this.debounceTimer = null;

        // Attach event listener
        this.input.addEventListener('input', (e) => this.handleSearch(e));

        // Handle enter key to select first result
        this.input.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') {
                this.selectFirstResult();
            }
        });
    }

    // Handle search input with debouncing
    handleSearch(event) {
        const query = event.target.value;

        // Debounce search for performance
        clearTimeout(this.debounceTimer);
        this.debounceTimer = setTimeout(() => {
            this.navigation.filter(query);
        }, 150);
    }

    // Select first visible result
    selectFirstResult() {
        const firstVisible = document.querySelector('.graph-link:not([style*="display: none"])');
        if (firstVisible) {
            firstVisible.click();
            this.input.blur();
        }
    }

    // Clear search
    clear() {
        this.input.value = '';
        this.navigation.filter('');
    }
}
