// Navigation component for the sidebar
export class Navigation {
    constructor(container, onGraphSelect) {
        this.container = container;
        this.onGraphSelect = onGraphSelect;
        this.currentGraphId = null;
    }

    // Set the active graph in the navigation
    setActive(graphId) {
        // Remove previous active state
        const prevActive = this.container.querySelector('.graph-link.active');
        if (prevActive) {
            prevActive.classList.remove('active');
        }

        // Set new active state
        const newActive = this.container.querySelector(`[data-graph-id="${graphId}"]`);
        if (newActive) {
            newActive.classList.add('active');
            // Scroll into view if needed
            newActive.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
        }

        this.currentGraphId = graphId;
    }

    // Attach click event listeners to graph links
    attachEventListeners() {
        const links = this.container.querySelectorAll('.graph-link');
        links.forEach(link => {
            link.addEventListener('click', (e) => {
                e.preventDefault();
                const graphId = link.dataset.graphId;
                const graphPath = link.dataset.graphPath;
                if (this.onGraphSelect) {
                    this.onGraphSelect(graphId, graphPath);
                }
            });
        });
    }

    // Initialize navigation (attach listeners)
    init() {
        this.attachEventListeners();
    }

    // Filter graphs based on search query
    filter(query) {
        const normalizedQuery = query.toLowerCase().trim();
        const links = this.container.querySelectorAll('.graph-link');

        if (!normalizedQuery) {
            // Show all
            links.forEach(link => {
                link.style.display = '';
            });
            this.updateCategoryCounts();
            return;
        }

        // Filter links
        links.forEach(link => {
            const title = link.querySelector('.graph-title').textContent.toLowerCase();
            const id = link.dataset.graphId.toLowerCase();

            if (title.includes(normalizedQuery) || id.includes(normalizedQuery)) {
                link.style.display = '';
            } else {
                link.style.display = 'none';
            }
        });

        this.updateCategoryCounts();
    }

    // Update category/domain counts based on visible items
    updateCategoryCounts() {
        // Update domain counts
        const domains = this.container.querySelectorAll('.domain');
        domains.forEach(domain => {
            const visibleLinks = domain.querySelectorAll('.graph-link:not([style*="display: none"])');
            const title = domain.querySelector('.domain-title');
            if (title) {
                const domainName = title.textContent.replace(/\s*\(\d+\)\s*$/, '');
                title.textContent = `${domainName} (${visibleLinks.length})`;
            }
            // Hide domain if no visible links
            domain.style.display = visibleLinks.length > 0 ? '' : 'none';
        });

        // Update category counts
        const categories = this.container.querySelectorAll('.category');
        categories.forEach(category => {
            const visibleDomains = category.querySelectorAll('.domain:not([style*="display: none"])');
            const title = category.querySelector('.category-title');
            if (title) {
                const categoryName = title.textContent.replace(/\s*\(\d+\)\s*$/, '');
                const totalVisible = Array.from(visibleDomains).reduce((sum, domain) => {
                    const links = domain.querySelectorAll('.graph-link:not([style*="display: none"])');
                    return sum + links.length;
                }, 0);
                title.textContent = `${categoryName} (${totalVisible})`;
            }
            // Hide category if no visible domains
            category.style.display = visibleDomains.length > 0 ? '' : 'none';
        });
    }
}
