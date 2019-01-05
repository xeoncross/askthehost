var gulp = require('gulp'),
	uglify = require('gulp-uglify'),
	rename = require('gulp-rename'),
	// mainBowerFiles = require('main-bower-files'),
	size = require('gulp-size'),
	// ngAnnotate = require('gulp-ng-annotate'),
	debug = require('gulp-debug'),
	concat = require('gulp-concat');


// Concatenate & Minify JS
gulp.task('scripts', function() {

	return gulp.src([
			// './socket.io.js',
			// './bower_components/jquery/dist/jquery.js',
			'./bower_components/angular-socket-io/socket.js',
			'./debounce.js',
			'./app.js'
		])
		.pipe(debug())
		// .pipe(ngAnnotate())
		.pipe(concat('main.js'))
		.pipe(gulp.dest('./'))
		// .pipe(rename('main.min.js'))
		.pipe(uglify({
			compress: {
				unused: false
			}
		}))
		.pipe(size())
		.pipe(gulp.dest('../static/'));
});

// gulp.task('css', function() {

// 	return gulp.src([
// 			// './bower_components/material-design-icons/iconfont/material-icons.css',
// 			'./bower_components/material-design-icons/iconfont/',
// 		])
// 		.pipe(size())
// 		.pipe(gulp.dest('../static/'));
// });


// gulp.task('build', ['scripts', 'css']);
gulp.task('default', ['scripts']);
