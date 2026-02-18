categories = {}

function escape(unsafe) {
    return $("<div/>").text(unsafe).html();
}

function fetchCategories($container) {
    $.ajax({
        url: "/api/categories",
        success: function (results) {
            results.forEach((result, _1, _2) => {
                const id = result.ID, name = result.DisplayName;
                categories[id] = name;
                const containerEl = $(`<div class="form-check form-check-inline"/>`)
                    .append($(`<input class="form-check-input" type="checkbox" value="${id}" id="category-${id}" name="categories[]" checked/>`))
                    .append($(`<label class="form-check-label" for="category-${id}">${escape(name)}</label>`));
                $container.append(containerEl);
            })
        }
    })
}

function wireImagePreview($input, $img) {
    $input.change(function () {
        if (this.files && this.files[0]) {
            const reader = new FileReader();
            reader.onload = function (e) {
                $img.attr('src', e.target.result);
            }
            reader.readAsDataURL(this.files[0]);
        }
    })
}

function wireSubmit($form, $resultsContainer) {
    $form.submit((event) => {
        event.preventDefault();
        $resultsContainer.html('')
        const data = new FormData($form[0]);
        $.ajax({
            method: "POST",
            url: "/api/search",
            data: data,
            processData: false,
            contentType: false,
            success: function (results) {
                results.forEach((result, _1, _2) => {
                    const name = result.DisplayName, tag = result.DisambiguationTag, score = result.SimilarityScore;
                    const display = !!tag ? `${name} (${tag})` : name;
                    const category = categories[result.CategoryID];
                    const containerEl = $(`<div class="search-result"/>`)
                        .append($(`<h3 class="h5"><a href="https://www.google.com/search?q=${escape(display)}+${escape(category)}">${escape(display)}</a</h3>`))
                        .append($(`<div>Category: ${category}</div>`))
                        .append($(`<div>Similarity Score: ${score.toFixed(2)}</div>`))
                    $resultsContainer.append(containerEl);
                })
            }
        })
    })
}

$(function () {
    fetchCategories($("#categories-container"));
    wireImagePreview($("#image-input"), $("#image-preview"))
    wireSubmit($("#search-form"), $("#results-container"))
})
