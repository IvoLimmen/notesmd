document.addEventListener('keydown', function (event) {
    if(event.key == 'e') {
        var location = window.location.toString().replace("/view/", "/edit/")
        window.location = location;
    }
});